package component

import (
	"net/http"
)

func NewServerStatusApp() *StatusCollection {
	cardinalStatusObj := StatusObject{
		statusName: "Cardinal",
		check: func(status *StatusObject) {
			resp, err := http.Get("localhost:8080/health")
			if err != nil {
				status.SetStatus(PENDING)
			} else {
				if resp.StatusCode == 200 {
					status.SetStatus(SUCCESS)
				} else {
					status.SetStatus(PENDING)
				}
			}
		},
	}
	cardinalStatusObj.SetStatus(PENDING)
	nakamaStatusObj := StatusObject{
		statusName: "Nakama",
		check: func(status *StatusObject) {
			resp, err := http.Get("localhost:8080/health")
			if err != nil {
				status.SetStatus(PENDING)
			} else {
				if resp.StatusCode == 200 {
					status.SetStatus(SUCCESS)
				} else {
					status.SetStatus(PENDING)
				}
			}
		},
	}
	nakamaStatusObj.SetStatus(PENDING)

	return NewStatusCollection([]*StatusObject{
		&cardinalStatusObj, &nakamaStatusObj,
	}, WithBorder)
}
