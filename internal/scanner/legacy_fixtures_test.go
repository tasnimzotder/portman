//go:build darwin

package scanner

import (
	"bufio"
	"github.com/tasnimzotder/portman/internal/model"
	"strconv"
	"strings"
)

// Adapt historical text fixtures to the shared binding builder. Production scans
// use parseLsofFields; regression_test.go covers that parser and actual sockets.
// Compatibility parser for saved human-readable lsof fixtures.
func parseLegacyRecords(out string) ([]socketRecord, error) {
	var records []socketRecord
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 9 || f[0] == "COMMAND" {
			continue
		}
		pid, err := strconv.Atoi(f[1])
		if err != nil {
			return nil, err
		}
		r := socketRecord{pid: pid, command: decodeLsofEscapes(f[0]), user: f[2], family: f[4], protocol: strings.ToLower(f[7]), name: f[8]}
		if len(f) > 9 {
			r.state = strings.Trim(f[9], "()")
		}
		records = append(records, r)
	}
	return records, sc.Err()
}
func (s *DarwinScanner) parseLsofOutput(out string) ([]model.Listener, error) {
	records, err := parseLegacyRecords(out)
	if err != nil {
		return nil, err
	}
	return buildListeners(records, true), nil
}
func (s *DarwinScanner) parsePortDetail(out string, port int) (*model.Listener, []model.Connection) {
	records, err := parseLegacyRecords(out)
	if err != nil {
		return nil, nil
	}
	for _, l := range buildListeners(records, true) {
		if l.Port == port {
			return &l, l.Connections
		}
	}
	return nil, nil
}
