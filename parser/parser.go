package parser

import (
	log "github.com/sirupsen/logrus"
)

func ParseCall(raw RawCall) (pc ParsedCall, err error) {
	pc.Set("departmentID", raw.Get("departmentID").Str)
	pc.Set("departmentName", raw.Get("departmentName").Str)
	pc.Set("id", raw.Get("uuid").Str)
	pc.Set("callType", raw.Get("callType").Str)
	start, err := getTime(raw.Get("start"))
	if err != nil {
		return nil, err
	}
	connect, err := getTime(raw.Get("connect"))
	if err != nil {
		return nil, err
	}
	disconnect, err := getTime(raw.Get("disconnect"))
	if err != nil {
		return nil, err
	}
	pc.Set("start", start)
	pc.SetDuration(start, connect, disconnect)
	if !connect.IsZero() {
		pc.Set("answered", true)
	}
	pc.Set("clientNumber", raw.Get("client").Uint())
	pc.Set("employeeNumber", raw.Get("employee.number").Uint())
	pc.Set("employeeName", raw.Get("employee.name").Str)
	log.Infof("resulted call: %s", string(pc))
	return pc, nil
}
