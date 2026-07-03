package custerr

import (
	"net/http"
)

var statusTextMap = map[int]string{
	http.StatusOK:                  "Success",
	http.StatusBadRequest:          "Invalid data request",
	http.StatusBadGateway:          "Oops something went wrong",
	http.StatusInternalServerError: "Oops something went wrong",
	http.StatusUnauthorized:        "Not authorized to access the service",
	http.StatusCreated:             "Resource has been created",
	http.StatusAccepted:            "Resource has been accepted",
	http.StatusForbidden:           "Forbidden access the resource",
	http.StatusProxyAuthRequired:   "The resource owner or authorization server denied the request",
	http.StatusNotFound:            "Not Found",
	http.StatusConflict:            "Conflict",
	http.StatusUnprocessableEntity: "Validation failed",
}

func StatusText(code int) string {
	if msg, ok := statusTextMap[code]; ok {
		return msg
	}
	return "Oops something went wrong"
}
