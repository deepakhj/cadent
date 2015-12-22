package prereg

import (
	"fmt"
	"regexp"
	"strings"
)

/********************** prefix filter ***********************/

type PrefixFilter struct {
	Prefix   string `json:"prefix"`
	IsReject bool   `json:"is_rejected"`
	backend  string `json:"backend"`
}

func (pref PrefixFilter) ToString() string {
	return fmt.Sprintf(
		"Type: `%-6s` Match:`%-30s` Rejected: `%v` Backend: `%s`",
		pref.Type(),
		pref.Name(),
		pref.Rejecting(),
		pref.Backend(),
	)
}

func (pref *PrefixFilter) Name() string {
	return pref.Prefix
}
func (pref *PrefixFilter) Type() string {
	return "prefix"
}
func (pref *PrefixFilter) Rejecting() bool {
	return pref.IsReject
}
func (pref *PrefixFilter) Init() error {
	return nil
}

func (pref *PrefixFilter) Match(in string) (bool, bool, error) {
	match := strings.HasPrefix(in, pref.Prefix)
	return match, pref.IsReject, nil
}

func (pref *PrefixFilter) Backend() string {
	return pref.backend
}

func (pref *PrefixFilter) SetBackend(back string) (string, error) {
	pref.backend = back
	return back, nil
}

/**********************   SubString filter ***********************/
type SubStringFilter struct {
	SubString string `json:"substring"`
	IsReject  bool   `json:"is_rejected"`
	backend   string `json:"backend"`
}

func (sfilter *SubStringFilter) Name() string {
	return sfilter.SubString
}
func (sfilter *SubStringFilter) Type() string {
	return "substring"
}
func (sfilter *SubStringFilter) Rejecting() bool {
	return sfilter.IsReject
}
func (sfilter *SubStringFilter) Init() error {
	return nil
}
func (sfilter *SubStringFilter) Match(in string) (bool, bool, error) {
	match := strings.Contains(in, sfilter.SubString)
	return match, sfilter.IsReject, nil
}

func (sfilter *SubStringFilter) Backend() string {
	return sfilter.backend
}

func (sfilter *SubStringFilter) SetBackend(back string) (string, error) {
	sfilter.backend = back
	return back, nil
}
func (sfilter *SubStringFilter) ToString() string {
	return fmt.Sprintf(
		"Type: `%-6s` Match:`%-30s` Rejected: `%v` Backend: `%s`",
		sfilter.Type(),
		sfilter.Name(),
		sfilter.Rejecting(),
		sfilter.Backend(),
	)
}

/**********************   reg filter ***********************/
type RegexFilter struct {
	RegexString string `json:"regex"`
	IsReject    bool   `json:"is_rejected"`
	backend     string `json:"backend"`

	thereg *regexp.Regexp
}

func (refilter *RegexFilter) Name() string {
	return refilter.RegexString
}
func (refilter *RegexFilter) Type() string {
	return "regex"
}
func (refilter *RegexFilter) Rejecting() bool {
	return refilter.IsReject
}
func (refilter *RegexFilter) Init() error {
	refilter.thereg = regexp.MustCompile(refilter.RegexString)
	return nil
}
func (refilter *RegexFilter) Match(in string) (bool, bool, error) {
	match := refilter.thereg.MatchString(in)
	return match, refilter.IsReject, nil
}

func (refilter *RegexFilter) Backend() string {
	return refilter.backend
}

func (refilter *RegexFilter) SetBackend(back string) (string, error) {
	refilter.backend = back
	return back, nil
}
func (refilter *RegexFilter) ToString() string {
	return fmt.Sprintf(
		"Type: `%-6s` Match:`%-30s` Rejected: `%v` Backend: `%s`",
		refilter.Type(),
		refilter.Name(),
		refilter.Rejecting(),
		refilter.Backend(),
	)
}

/**********************  filter list ***********************/

type PreReg struct {
	DefaultBackEnd string `json:"default_backend"`
	Name           string `json:"name"`

	// the actual "listening" server that this reg is pinned to
	ListenServer string `json:listen_server_name`

	FilterList []FilterItem `json:"map"`
}

func (pr *PreReg) MatchingFilters(line string) []FilterItem {
	// just the list of filters that match a string

	var fitems = make([]FilterItem, 0)
	for _, fil := range pr.FilterList {
		matched, _, _ := fil.Match(line)
		if matched {
			fitems = append(fitems, fil)
		}
	}
	return fitems
}

func (pr *PreReg) FirstMatchFilter(line string) (FilterItem, bool, error) {
	for _, fil := range pr.FilterList {
		matched, reject, err := fil.Match(line)
		if matched {
			return fil, reject, err
		}
	}
	return nil, false, nil
}

func (pr *PreReg) FirstMatchBackend(line string) (string, bool, error) {
	for _, fil := range pr.FilterList {
		matched, reject, err := fil.Match(line)
		if matched {
			return fil.Backend(), reject, err
		}
	}
	return pr.DefaultBackEnd, false, nil
}

func (pr *PreReg) LogConfig() {
	log.Debug(" - Pre Filter Group: %s", pr.Name)
	log.Debug("   - Pinned To Listener: %s", pr.ListenServer)
	for idx, filter := range pr.FilterList {
		log.Debug("   - PreFilter %2d:: %s", idx, filter.ToString())
	}
}

/**********************  maps of filter list ***********************/

// list of filters
type PreRegMap map[string]*PreReg

func (lpr *PreRegMap) MatchingFilters(line string) []FilterItem {
	var fitems = make([]FilterItem, 0)
	for _, pr := range *lpr {
		gots := pr.MatchingFilters(line)
		if gots != nil {
			fitems = append(fitems, gots...)
		}
	}
	return fitems
}

func (lpr *PreRegMap) FirstMatchingFilters(line string) []FilterItem {
	var fitems = make([]FilterItem, 0)
	for _, pr := range *lpr {
		gots := pr.MatchingFilters(line)
		if gots != nil {
			fitems = append(fitems, gots...)
		}
	}
	return fitems
}
func (lpr *PreRegMap) FirstMatchBackends(line string) ([]string, []bool, []error) {
	var backs = make([]string, 0)
	var reject = make([]bool, 0)
	var errs = make([]error, 0)
	for _, pr := range *lpr {
		bck, rjc, err := pr.FirstMatchBackend(line)
		backs = append(backs, bck)
		reject = append(reject, rjc)
		errs = append(errs, err)
	}
	return backs, reject, errs
}

func (lpr *PreRegMap) LogConfig() {
	log.Debug("=== Pre Filters ===")
	for _, filter := range *lpr {
		filter.LogConfig()
	}
}
