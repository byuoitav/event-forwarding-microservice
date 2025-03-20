package managers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/byuoitav/common/nerr"
	sd "github.com/byuoitav/common/state/statedefinition"
	"github.com/byuoitav/event-forwarding-microservice/couch"
)

// CouchStaticDevice is just an sd StaticDevice with an _id and a _rev
type CouchStaticDevice struct {
	sd.StaticDevice
	Rev
}

// Rev is a utility struct used for updating revisions
type Rev struct {
	Revision string `json:"_rev,omitempty"`
	ID       string `json:"_id"`
}

// GetDefaultCouchDeviceBuffer starts and returns a buffer manager
func GetDefaultCouchDeviceBuffer(couchaddr, database string, interval time.Duration) *CouchDeviceBuffer {
	//we'll need to initialize from the server
	val := &CouchDeviceBuffer{
		incomingChannel:    make(chan sd.StaticDevice, 10000),
		reingestionChannel: make(chan CouchStaticDevice, 1000),
		revChannel:         make(chan []Rev, 100),

		curBuffer: make(map[string]CouchStaticDevice), revBuffer: make(map[string]Rev),

		interval:  interval,
		database:  database,
		couchaddr: couchaddr,
	}

	go val.start()
	return val
}

// CouchDeviceBuffer takes a static device and buffers them for storage in couch
type CouchDeviceBuffer struct {
	incomingChannel    chan sd.StaticDevice
	reingestionChannel chan CouchStaticDevice
	revChannel         chan []Rev

	curBuffer map[string]CouchStaticDevice
	revBuffer map[string]Rev

	interval  time.Duration
	database  string
	couchaddr string
}

// Send fulfils the manager interface
func (c *CouchDeviceBuffer) Send(toSend interface{}) error {

	dev, ok := toSend.(sd.StaticDevice)
	if !ok {
		return nerr.Create("Invalid type, couch device buffer expects a StaticDevice", "invalid-type")
	}

	c.incomingChannel <- dev

	return nil
}

func (c *CouchDeviceBuffer) start() {

	slog.Info("Starting couch buffer for database", "database", c.database)
	ticker := time.NewTicker(c.interval)

	for {
		select {
		case <-ticker.C:
			//send it off
			slog.Debug("Sending bulk ELK update", "database", c.database)

			//send the current one
			sendBulkDeviceUpdate(c.curBuffer, c.revChannel, c.reingestionChannel, c.couchaddr, c.database)

			//create a fresh buffer
			c.curBuffer = make(map[string]CouchStaticDevice)

		case dev := <-c.incomingChannel:
			slog.Debug("Received device", "device", dev)
			c.buffer(dev)

		case revs := <-c.revChannel:
			slog.Debug("Updating revision numbers")
			c.updateRevs(revs)
		case redo := <-c.reingestionChannel:
			//just dump it in, it's updated
			c.curBuffer[redo.DeviceID] = redo
			c.revBuffer[redo.DeviceID] = redo.Rev
		}
	}
}

func (c *CouchDeviceBuffer) reingest(dev CouchStaticDevice) {

	//check to see if we've gotten a new update
	if v, ok := c.curBuffer[dev.DeviceID]; ok {
		v.Rev = dev.Rev

		c.curBuffer[dev.DeviceID] = v
	} else {
		c.curBuffer[dev.DeviceID] = dev
	}

	c.revBuffer[dev.DeviceID] = dev.Rev
}

func (c *CouchDeviceBuffer) buffer(dev sd.StaticDevice) {

	//check to see if it's in the cur buffer, if not, get it's _rev from the revBuffer
	if v, ok := c.curBuffer[dev.DeviceID]; ok {
		v.StaticDevice = dev
		c.curBuffer[dev.DeviceID] = v

		return
	}

	//check the rev table
	if v, ok := c.revBuffer[dev.DeviceID]; ok {
		c.curBuffer[dev.DeviceID] = CouchStaticDevice{
			Rev:          v,
			StaticDevice: dev,
		}

		return
	}

	//it's not in either, we just create a new entry for it, set the ID, leave rev blank
	c.curBuffer[dev.DeviceID] = CouchStaticDevice{
		StaticDevice: dev,
		Rev:          Rev{ID: dev.DeviceID},
	}
}

func (c *CouchDeviceBuffer) updateRevs(r []Rev) {
	for i := range r {
		if v, ok := c.curBuffer[r[i].ID]; ok {
			v.Revision = r[i].Revision
			c.curBuffer[r[i].ID] = v
			return
		}

		//update the rev table
		c.revBuffer[r[i].ID] = r[i]
	}
}

// CouchStaticUpdateBody is a utility to marshal the structure that couch bulk API expects
type CouchStaticUpdateBody struct {
	Docs []CouchStaticDevice `json:"docs"`
}

// CouchBulkUpdateResponse is the couch reseponse to bulk update/create requests
type CouchBulkUpdateResponse struct {
	OK       bool   `json:"ok,omitempty"`
	ID       string `json:"id,omitempty"`
	Revision string `json:"rev,omitempty"`
	Error    string `json:"error,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// assume is run in go routine
func sendBulkDeviceUpdate(toSend map[string]CouchStaticDevice, returnChan chan<- []Rev, reingestionChannel chan<- CouchStaticDevice, addr, database string) {

	if len(toSend) < 1 {
		slog.Info("No devices to send, returning", "addr", addr, "database", database)
		return
	}
	slog.Info("Sending bulk update", "addr", addr, "database", database)

	//go through the map and create the CouchStaticUpdateBody
	body := CouchStaticUpdateBody{}

	for _, v := range toSend {
		body.Docs = append(body.Docs, v)
	}

	resp, err := couch.MakeRequest(
		fmt.Sprintf("%v/%v/_bulk_docs", strings.Trim(addr, "/"), database),
		"POST",
		body,
	)
	if err != nil {
		if err.Type == http.StatusText(417) {
			//at least one device failed
			slog.Warn("At least one document in the bulk update failed")
		}
		slog.Error("Bad response received from Couch", "error", err.Error())
		return
	}
	//we unmarshal the response into the update respons
	var respArray []CouchBulkUpdateResponse

	er := json.Unmarshal(resp, &respArray)
	if er != nil {
		slog.Error("Unknown response received", "error", er.Error(), "response", string(resp))
		return
	}
	toReturn := []Rev{}

	toBeFixed := make(map[string]CouchStaticDevice)

	//we need to go through and either a) update the resp array b) if there was an error and the error was document_conflict
	for _, cur := range respArray {
		//check status, update or requque as needed.
		if cur.OK {
			toReturn = append(toReturn, Rev{ID: cur.ID, Revision: cur.Revision})
		} else {
			if cur.Error == "conflict" {
				//we need to get the new revision, then dump it back up the reingestion channel
				slog.Warn("Document update conflict, rectifying", "id", cur.ID)
				toBeFixed[cur.ID] = toSend[cur.ID]
				continue
			} else {
				slog.Error("Couldn't create/update document", "id", cur.ID, "error", cur.Error, "reason", cur.Reason)
				continue
			}
		}
	}

	go getUpdatedRevs(addr, database, toBeFixed, reingestionChannel)
	returnChan <- toReturn
}

// CouchBulkRequestItem .
type CouchBulkRequestItem struct {
	ID string `json:"id"`
}

// CouchBulkDocumentRequest .
type CouchBulkDocumentRequest struct {
	Docs []CouchBulkRequestItem `json:"docs"`
}

// CouchBulkDevicesResponse .
type CouchBulkDevicesResponse struct {
	Results []struct {
		ID   string `json:"id"`
		Docs []struct {
			OK    CouchStaticDevice       `json:"ok"`
			Error CouchBulkUpdateResponse `json:"error"`
		} `json:"docs"`
	} `json:"results"`
}

func getUpdatedRevs(addr, database string, toBeFixed map[string]CouchStaticDevice, reingestionChannel chan<- CouchStaticDevice) {
	if len(toBeFixed) < 1 {
		slog.Debug("No updated revisions to get, returning")
		return
	}

	b := CouchBulkDocumentRequest{}
	//we do a bulk request for all the ids
	for i := range toBeFixed {
		b.Docs = append(b.Docs, CouchBulkRequestItem{ID: toBeFixed[i].DeviceID})
	}

	resp, err := couch.MakeRequest(
		fmt.Sprintf("%v/%v/_bulk_get", strings.Trim(addr, "/"), database),
		"POST",
		b,
	)
	if err != nil {
		slog.Debug("Couldn't complete bulk request to update couch revision numbers", "error", err.Error())
		return
	}

	var rr CouchBulkDevicesResponse
	er := json.Unmarshal(resp, &rr)
	if er != nil {
		slog.Error("Unknown response received", "error", er.Error(), "response", string(resp))
		return
	}

	for i := range rr.Results {
		//send it down reingestion channel
		if len(rr.Results[i].Docs[0].OK.DeviceID) > 0 {
			v := toBeFixed[rr.Results[i].Docs[0].OK.DeviceID]
			v.Rev = rr.Results[i].Docs[0].OK.Rev

			reingestionChannel <- v
		} else {
			slog.Error("Unknown key requested while getting updated revisions", "id", rr.Results[i].ID, "error", rr.Results[i].Docs[0].Error.Error, "reason", rr.Results[i].Docs[0].Error.Reason)
		}
	}
}
