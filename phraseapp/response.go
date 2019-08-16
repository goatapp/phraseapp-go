package phraseapp

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const docsURL = `https://developers.phrase.com/api/`

func further() string {
	return fmt.Sprintf("\nFor further information see:\n%s", docsURL)
}

func handleResponseStatus(resp *http.Response, expectedStatus int) error {
	switch status := resp.StatusCode; status {
	case expectedStatus:
		return nil
	case http.StatusBadRequest:
		e := new(ErrorResponse)
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return err
		}
		return e
	case http.StatusUnauthorized:
		return fmt.Errorf("%d - %s\nThe credentials you provided are invalid.%s", status, http.StatusText(status), further())
	case http.StatusForbidden:
		return fmt.Errorf("%d - %s\nYou are not authorized to perform the requested action on the requested resource. Check if your provided access_token has the correct scope.%s", status, http.StatusText(status), further())
	case http.StatusNotFound:
		var rsp struct {
			Message string `json:"message"`
		}
		b, _ := ioutil.ReadAll(resp.Body)
		decodeErr := json.Unmarshal(b, &rsp)
		if decodeErr != nil {
			return ErrNotFound{Message: fmt.Sprintf("%d - Resource Not Found\nThe resource you requested or referenced resources you required do either not exist or you do not have the authorization to request this resource.", status)}
		}
		return ErrNotFound{Message: string(b)}
	case http.StatusUnsupportedMediaType, http.StatusUnprocessableEntity:
		e := new(ValidationErrorResponse)
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return err
		}
		return e
	case http.StatusTooManyRequests:
		e, err := NewRateLimitError(resp)
		if err != nil {
			return err
		}
		return e
	default:
		return fmt.Errorf("Unexpected HTTP Status Code (%d %s) received; expected %d %s.%s", status, http.StatusText(status), expectedStatus, http.StatusText(expectedStatus), further())
	}
}
