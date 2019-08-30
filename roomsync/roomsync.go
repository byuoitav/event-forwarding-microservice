package roomsync

import (
	"context"
	"strings"
	"time"

	"github.com/byuoitav/common/db"
	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	sd "github.com/byuoitav/common/state/statedefinition"
	"github.com/byuoitav/common/structs"
	"github.com/byuoitav/event-forwarding-microservice/cache"
	"github.com/fatih/color"
)

func init() {
}

//StartRoomSync will pull config information from the config db to store in ELK on the interval defined. Ctx is for cancellation.
func StartRoomSync(interval time.Duration, ctx context.Context) {
	log.L.Infof(color.HiBlueString("Starting room sync"))

	run()

	//run an initial one
	t := time.NewTicker(interval)
	for {
		select {
		case <-t.C:
			run()

		case <-ctx.Done():
			log.L.Infof("Cancel called, ending room sync.")
			return

		}
	}
}

func run() {
	rms, err := GetRoomsWithTypes()
	if err != nil {
		log.L.Errorf("Couldn't sync room information: %v", err.Error())
		return
	}

	for i := range rms {
		_, _, err := cache.GetCache("default").CheckAndStoreRoom(rms[i])
		if err != nil {
			log.L.Errorf("Couldn't sync room information for room %v: %v", rms[i].RoomID, err.Error())
			continue
		}
	}
}

//GetRoomsWithTypes .
func GetRoomsWithTypes() ([]sd.StaticRoom, *nerr.E) {

	toReturn := map[string]sd.StaticRoom{}

	//start with the DMPS systems
	list, err := GetDMPSSystems()
	if err != nil {
		return []sd.StaticRoom{}, err.Addf("Couldn't build list of rooms with system types")
	}

	for i := range list {
		//build our room
		cur := sd.StaticRoom{
			RoomID:      list[i],
			BuildingID:  strings.Split(list[i], "-")[0],
			Designation: sd.Production,
			SystemType:  []string{sd.DMPS},
			UpdateTimes: map[string]time.Time{
				"system-type": time.Now(),
				"designation": time.Now(),
			},
		}
		toReturn[list[i]] = cur
	}

	//we've done our DMPS stuff, now on to the regular stuff
	devs, er := db.GetDB().GetAllDevices()
	if er != nil {
		return []sd.StaticRoom{}, nerr.Translate(er).Addf("Couldn't build list of rooms with system types")
	}

	for i := range devs {
		//check the type against our map
		if v, ok := TypeMappings[devs[i].Type.ID]; ok {

			//check to see if we already have this room
			rmID := devs[i].GetDeviceRoomID()
			rm, ok := toReturn[rmID]
			if ok {
				if !rm.HasSystemType(v) {
					rm.SystemType = append(rm.SystemType, v)
				}
				toReturn[rmID] = rm
			} else {
				//create our new room
				cur := sd.StaticRoom{
					RoomID:     rmID,
					BuildingID: strings.Split(rmID, "-")[0],
					SystemType: []string{v},
					UpdateTimes: map[string]time.Time{
						"system-type": time.Now(),
					},
				}
				toReturn[rmID] = cur
			}
		}
	}

	roomsMap := map[string]structs.Room{}
	//get all the rooms:
	rms, er := db.GetDB().GetAllRooms()
	if er != nil {
		log.L.Errorf("Couldn't get all rooms: %v", er.Error())
	} else {
		for i := range rms {
			roomsMap[rms[i].ID] = rms[i]
		}
	}

	tmp := []sd.StaticRoom{}
	//we need to go through and do our designations too?
	for _, v := range toReturn {
		//check the designation
		if v.Designation == "" {
			if rm, ok := roomsMap[v.RoomID]; ok {
				v.Designation = rm.Designation
			}
			v.UpdateTimes["designation"] = time.Now()
		}

		tmp = append(tmp, v)
	}

	return tmp, nil
}

//GetDMPSSystems .
func GetDMPSSystems() ([]string, *nerr.E) {

	log.L.Debugf("Getting the list of DMPS systems")

	//from couch we just check the 'dmps' database for the list, it contains the list of devices, we build a room for each unique oneA
	rooms := map[string]bool{}
	toSend := []string{}

	list, er := db.GetDB().GetDMPSList()
	if er != nil {
		return toSend, nerr.Translate(er).Addf("Couldn't get list of DMPS systems")
	}

	for i := range list.List {
		splitName := strings.Split(list.List[i].Hostname, "-")
		if len(splitName) != 3 {
			log.L.Error("Bad DMPS system hostname %v", list.List[i].Hostname)
			continue
		}

		rooms[splitName[0]+"-"+splitName[1]] = true
	}

	for k := range rooms {
		toSend = append(toSend, k)
	}

	log.L.Debugf("Found %v rooms with DMPS systems", len(toSend))
	return toSend, nil
}

//TypeMappings means that, by room, if any devices in the room have these devices we add these system types to the room record.
var TypeMappings map[string]string

func init() {
	TypeMappings = map[string]string{
		"DMPS":            sd.DMPS,
		"Pi3":             sd.Pi,
		"SchedulingPanel": sd.Scheduling,
		"Timeclock":       sd.Timeclock,
	}
}
