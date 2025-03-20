package shared

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/byuoitav/common/nerr"
	sd "github.com/byuoitav/common/state/statedefinition"
)

// GetNewRoom .
func GetNewRoom(id string) (sd.StaticRoom, *nerr.E) {

	rm := strings.Split(id, "-")
	if len(rm) != 2 {
		slog.Error("Invalid Room", "id", id)
		return sd.StaticRoom{}, nerr.Create(fmt.Sprintf("Can't build device manager: invalid ID %v", id), "invalid-id")
	}

	room := sd.StaticRoom{
		RoomID:      id,
		BuildingID:  rm[0],
		UpdateTimes: make(map[string]time.Time),
	}
	return room, nil
}
