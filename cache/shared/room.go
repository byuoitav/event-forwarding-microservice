package shared

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	//sd "github.com/byuoitav/common/state/statedefinition"
	customerror "github.com/byuoitav/event-forwarding-microservice/error"
	sd "github.com/byuoitav/event-forwarding-microservice/statedefinition"
)

// GetNewRoom .
func GetNewRoom(id string) (sd.StaticRoom, error) {

	rm := strings.Split(id, "-")
	if len(rm) != 2 {
		errorLog := fmt.Sprintf("Invalid Room %v", id)
		slog.Error(errorLog)
		rErr := &customerror.StandardError{
			Message: fmt.Sprintf("Can't build device manager: invalid ID %v", id),
		}
		return sd.StaticRoom{}, rErr
	}

	room := sd.StaticRoom{
		RoomID:      id,
		BuildingID:  rm[0],
		UpdateTimes: make(map[string]time.Time),
	}
	return room, nil
}
