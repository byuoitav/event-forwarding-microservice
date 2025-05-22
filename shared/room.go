package shared

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	sd "github.com/byuoitav/event-forwarding-microservice/state/statedefinition"
)

// GetNewRoom .
func GetNewRoom(id string) (sd.StaticRoom, error) {

	rm := strings.Split(id, "-")
	if len(rm) != 2 {
		slog.Error("Invalid Room", "id", id)
		return sd.StaticRoom{}, fmt.Errorf("can't build device manager: invalid ID %v", id)
	}

	room := sd.StaticRoom{
		RoomID:      id,
		BuildingID:  rm[0],
		UpdateTimes: make(map[string]time.Time),
	}
	return room, nil
}
