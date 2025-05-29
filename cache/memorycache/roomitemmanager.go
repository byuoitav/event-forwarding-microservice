package memorycache

import (
	"errors"
	"log/slog"
	"time"

	sd "github.com/byuoitav/event-forwarding-microservice/state/statedefinition"
)

/*RoomItemManager handles managing access to a single room in a cache. Changes to the room are submitted via the IncomingWriteChan and reads are submitted via the IncomingReadChan.
 */
type RoomItemManager struct {
	WriteRequests chan RoomTransactionRequest //channel to buffer changes to the room.
	ReadRequests  chan chan sd.StaticRoom
	KillChannel   chan bool
}

// RoomTransactionRequest is submitted to read/write a the room being managed by this manager
type RoomTransactionRequest struct {
	ResponseChan chan RoomTransactionResponse
	MergeRoom    sd.StaticRoom
}

// RoomTransactionResponse .
type RoomTransactionResponse struct {
	Changes bool          //if the Transaction Request resulted in changes
	NewRoom sd.StaticRoom //the updated room with the changes included in the Transaction request included
	Error   error         //if there were errors
}

// GetNewRoomManager .
func GetNewRoomManager(id string) RoomItemManager {
	a := RoomItemManager{
		WriteRequests: make(chan RoomTransactionRequest, 100),
		ReadRequests:  make(chan chan sd.StaticRoom, 100),
	}

	F := false

	room := sd.StaticRoom{
		RoomID:          id,
		UpdateTimes:     make(map[string]time.Time),
		MaintenenceMode: &F,
	}

	go StartRoomManager(a, room)
	return a
}

// StartRoomManager .
func StartRoomManager(m RoomItemManager, room sd.StaticRoom) {
	var merged sd.StaticRoom
	var changes bool
	var err error

	for {
		select {
		case <-m.KillChannel:
			slog.Info("Killing room manager", "roomID", room.RoomID)
			return
		case write := <-m.WriteRequests:
			if write.MergeRoom.RoomID != room.RoomID {
				write.ResponseChan <- RoomTransactionResponse{Error: errors.New("Can't change the ID of a room"), NewRoom: room, Changes: false}
				continue
			}
			_, merged, changes, err = sd.CompareRooms(room, write.MergeRoom)
			if err != nil {
				write.ResponseChan <- RoomTransactionResponse{Error: err, Changes: false}
				continue
			}
			if changes {
				room = merged
			}
			write.ResponseChan <- RoomTransactionResponse{Error: err, NewRoom: room, Changes: changes}
		case read := <-m.ReadRequests:
			read <- room
		}
	}
}
