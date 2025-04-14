package orm

import (
	"fmt"
	"time"

	"github.com/medatechnology/goutil/print"
	"github.com/medatechnology/goutil/timedate"
)

// Default status return, depends on the implementation, some information might be empty.
// But for first implementation of RQLite it should have almost all the information

// Struct to use as per-node status, information mostly from SettingsTable, but this is used for response
type StatusStruct struct {
	URL        string        `json:"url,omitempty"          db:"url"`         // URL (host + port)
	Version    string        `json:"version,omitempty"      db:"version"`     // version of the DMBS
	DBMS       string        `json:"dbms,omitempty"         db:"dbms"`        // version of the DMBS
	DBMSDriver string        `json:"dbms_driver,omitempty"  db:"dbms_driver"` // version of the DMBS
	StartTime  time.Time     `json:"start_time,omitempty"   db:"start_time"`
	Uptime     time.Duration `json:"uptime,omitempty"       db:"uptime"`
	DirSize    int64         `json:"dir_size,omitempty"     db:"dir_size"` // if applicable
	DBSize     int64         `json:"db_size,omitempty"      db:"db_size"`  // if applicable
	NodeID     string        `json:"node_id,omitempty"      db:"node_id"`  // DBMS node ID, was rqlite node_id from status
	IsLeader   bool          `json:"is_leader,omitempty"    db:"is_leader"`
	Leader     string        `json:"leader,omitempty"       db:"leader"`      // complete address (including protocol, ie: https://...)
	LastBackup time.Time     `json:"last_backup,omitempty"  db:"last_backup"` // if applicable
	Mode       string        `json:"mode,omitempty"         db:"mode"`        // options are r, w, or rw
	Nodes      int           `json:"nodes,omitempty"        db:"nodes"`       // total number of nodes in the cluster
	NodeNumber int           `json:"node_number,omitempty"  db:"node_number"` // this node number, actually this is not applicable in rqlite, because NodeID is string
	MaxPool    int           `json:"max_pool,omitempty"     db:"max_pool"`    // if applicable

}

// NodeStatusStruct is a struct that contains the status of a node, including its peers (if has peers)
// It is mostly derived from the SettingsTable but is used for response.
// Mode is : r , w, or rw (for read only, write only and read and write)
// Example of how to use it:
//
//	var nodeStatus NodeStatusStruct
//	nodeStatus.StatusStruct = StatusStruct{...}
//	nodeStatus.Peers = map[int]StatusStruct{...}
//
// Output:
//
//	{
//	  "url": "http://localhost:4001",
//	  "version": "3.5.0",
//	  "start_time": "2022-01-01T00:00:00Z",
//	  "uptime": "24h0m0s",
//	  "dir_size": 1024,
//	  "db_size": 2048,
//	  "node_id": "node1",
//	  "is_leader": true,
//	  "leader": "node1",
//	  "last_backup": "2022-01-01T00:00:00Z",
//	  "mode": "standalone",
//	  "nodes": 1,
//	  "node_number": 1,
//	  "peers": {
//	    2: {
//	      "url": "http://localhost:4002",
//	      "version": "3.5.0",
//	      "start_time": "2022-01-01T00:00:00Z",
//	      "uptime": "24h0m0s",
//	      "dir_size": 1024,
//	      "db_size": 2048,
//	      "node_id": "node2",
//	      "is_leader": false,
//	      "leader": "node1",
//	      "last_backup": "2022-01-01T00:00:00Z",
//	      "mode": "r",
//	      "nodes": 2,
//	      "node_number": 2
//	    }
//	  }
//	}
type NodeStatusStruct struct {
	StatusStruct
	Peers map[int]StatusStruct // all peers including the leader
}

// Output: Label: Host:Port [(Leader)|Empty] rw (1/3)
// func (s StatusStruct) String() string {
// 	leader := ""
// 	if s.IsLeader {
// 		leader = "(Leader)"
// 	}
// 	// return fmt.Sprintf("%s: %s %v %s (%d/%d)", s.Label, s.URL, leader, s.Mode, s.NodeNumber, s.Nodes)
// 	return fmt.Sprintf("%s: %v %s (%s/%d)", s.URL, leader, s.Mode, s.NodeID, s.Nodes)
// }

// This function is used to print the status of a node in a pretty format, mainly for debugging and logging purposes.
// Example usage: nodeStatus.PrintPretty("", "Node Status")
// Output: A formatted string representation of the node's status, including URL, version, start time, uptime, directory size, database size, node ID, leadership status, leader ID, last backup time, mode, number of nodes, and node number.
// This is mainly for debugging and logging
func (s *StatusStruct) PrintPretty(indent, title string) {
	if title == "" {
		title = "Status"
	}
	fmt.Println(title + ":")
	uptime := timedate.DurationUptimeShort(s.Uptime)
	if uptime == "" {
		uptime = "less than a minute"
	}
	fields := []struct {
		label string
		value string
	}{
		{"URL", s.URL},
		{"Version", s.Version},
		{"Start Time", s.StartTime.Format("2006-01-02 15:04:05")},
		{"Uptime", uptime},
		// {"Uptime", timedate.DurationUptimeShort(s.Uptime)},
		{"Dir Size", print.BytesToHumanReadable(s.DirSize, " ")},
		{"DB Size", print.BytesToHumanReadable(s.DBSize, " ")},
		{"Node ID", s.NodeID},
		{"Is Leader", fmt.Sprintf("%t", s.IsLeader)},
		{"Leader", s.Leader},
		{"Last Backup", s.LastBackup.Format("2006-01-02 15:04:05")},
		{"Mode", s.Mode},
		{"Nodes", fmt.Sprintf("%d", s.Nodes)},
		{"Node Number", fmt.Sprintf("%d", s.NodeNumber)},
	}

	maxLabelLength := 0
	for _, field := range fields {
		if len(field.label) > maxLabelLength {
			maxLabelLength = len(field.label)
		}
	}

	for _, field := range fields {
		if field.value != "" && field.value != "0001-01-01 00:00:00" && field.value != "0 B" && field.value != "less than a minute" {
			fmt.Printf("%s%-*s: %s\n", indent, maxLabelLength, field.label, field.value)
		}
	}
}

// func (s *NodeStatusStruct) Strings() []string {
// 	var status []string
// 	status = append(status, s.StatusStruct.String())
// 	for _, p := range s.Peers {
// 		status = append(status, p.String())
// 	}
// 	return status
// }

// This function is used to print the status of the node in a pretty format, including its peers.
// Example of how to use it: s.PrintPretty()
// Output: A formatted string representation of the node's status and its peers.
// This is mainly for debugging and logging
func (s *NodeStatusStruct) PrintPretty() {
	s.StatusStruct.PrintPretty("  ", "Status")
	for i := range s.Peers {
		p := s.Peers[i]
		if p.NodeNumber != s.NodeNumber {
			p.PrintPretty("    ", fmt.Sprintf("  Peers %d", i))
		}
	}
}
