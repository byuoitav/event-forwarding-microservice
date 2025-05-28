package shared

import (
	"fmt"
	"strings"
	"time"

	sd "github.com/byuoitav/event-forwarding-microservice/state/statedefinition"
)

// GetNewRoom .
func GetNewRoom(id string) (sd.StaticRoom, error) {
	rm := strings.Split(id, "-")
	if len(rm) != 2 {
		return sd.StaticRoom{}, fmt.Errorf("Can't build device manager: invalid ID %v", id)
	}

	room := sd.StaticRoom{
		RoomID:      id,
		BuildingID:  rm[0],
		UpdateTimes: make(map[string]time.Time),
	}
	return room, nil
}
