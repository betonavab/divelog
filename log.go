// Log package parses dive logs and provides operations to display, return or operated
// on divelog files as for example Sherwater produced XML files
package divelog

import (
	"bufio"
	"encoding/xml"
	"github.com/betonavab/deco"
	"fmt"
	"io"
	"math"
	"os"
	"time"
)

// DiveLog is the main interface for accesing a dive log. There could be multiple
// types of dive logs from different companies or different formats
type DiveLog interface {
	PrintAll(w io.Writer)
	PrintHisto(w io.Writer)

	FindMaxDepth() float64
	FindBestMatch(target time.Time, adjust1 int) (float64, bool)

	PlayIt(m deco.Model, usePPO2 bool) (max float64, min float64)
}

type SWLogRecord struct {
	XMLName        xml.Name `xml:"diveLogRecord"`
	Time           int      `xml:"currentTime"`
	Depth          float64  `xml:"currentDepth"`
	FirstStopDepth int      `xml:"firstStopDepth"`
	TTSMins        int      `xml:"ttsMins"`
	AveragePPO2    float64  `xml:"averagePPO2"`
	FractionO2     float64  `xml:"fractionO2"`
	FractionHe     float64  `xml:"fractionHe"`
	FirstStopTime  int      `xml:"firstStopTime"`
}

type SWLogRecords struct {
	XMLName       xml.Name      `xml:"diveLogRecords"`
	DiveLogRecord []SWLogRecord `xml:"diveLogRecord"`
}

type SWLog struct {
	XMLName   xml.Name `xml:"diveLog"`
	Number    int      `xml:"number"`
	GFMin     int      `xml:"gfMin"`
	GFMax     int      `xml:"gfMax"`
	Imperial  bool     `xml:"imperialUnits"`
	StartDate string   `xml:"startDate"`
	MaxDepth  int      `xml:"maxDepth"`
	MaxTime   int      `xml:"maxTime"`
	EndDate   string   `xml:"endDate"`

	DiveLogRecords SWLogRecords `xml:"diveLogRecords"`

	startdate time.Time
}

type SWDive struct {
	XMLName xml.Name `xml:"dive"`
	Version int      `xml:"version,attr"`
	DiveLog SWLog    `xml:"diveLog"`
}

var debug bool
var dwriter io.Writer

var pmodel bool
var mwriter io.Writer

// EnableDebug turn debugging on
func EnableDebug(w io.Writer) {
	debug = true
	dwriter = w
}

// DisableDebug turn debugging off
func DisableDebug() {
	debug = false
}

// EnablePmodel turn printing of model on
func EnablePmodel(w io.Writer) {
	pmodel = true
	mwriter = w
}

// DisablePmodel turn printing of model off
func DisablePmodel() {
	pmodel = false
}

func (r SWLogRecord) String() string {
	return fmt.Sprintf("record time=%v, depth=%v",
		r.Time, r.Depth)
}

func (l SWLog) String() string {
	return fmt.Sprintf("log number=%v, maxdepth=%v, maxtime=%v, from %v to %v",
		l.Number, l.MaxDepth, l.MaxTime, l.StartDate, l.EndDate)
}

// NewShearwaterLog creates a Log for an XML Shearwater log file
func NewShearwaterLog(name string) (*SWDive, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)
	d := xml.NewDecoder(r)

	var dive SWDive
	err = d.Decode(&dive)
	if err != nil {
		return nil, fmt.Errorf("failed to Decode: %v",err)
	}

	t, err := time.Parse(time.ANSIC+" UTC", dive.DiveLog.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid StartDate %v: %v",dive.DiveLog.StartDate,err)
	}
	dive.DiveLog.startdate = t

	return &dive, nil
}

// PrintAll rolls through the log and display the most important data for each entry
func (d *SWDive) PrintAll(w io.Writer) {
	fmt.Fprintln(w, d.DiveLog)
	for _, lr := range d.DiveLog.DiveLogRecords.DiveLogRecord {
		seconds := lr.Time
		t := d.DiveLog.startdate.Add(time.Duration(seconds) * time.Second)
		fmt.Fprintf(w, "%v depth %v ppo2 %v mix %v/%v\n", t.Format(time.UnixDate), lr.Depth,
			lr.AveragePPO2, lr.FractionO2, lr.FractionHe)
	}
}

func unit2min(u int) int {
	return int(float64(u) * (1.0 / 6.0))
}

// PrintHisto display an histogram of the log's profile
func (d *SWDive) PrintHisto(w io.Writer) {

	if d == nil {
		return
	}

	histo := make([]int, 1000)
	for _, lr := range d.DiveLog.DiveLogRecords.DiveLogRecord {
		i := int(math.Round(lr.Depth/10) * 10)
		if i < len(histo) {
			histo[i]++
		}
	}

	n := 0
	m := 0
	for i, h := range histo {
		if h == 0 {
			continue
		}
		n += i * h
		m += h
	}
	fmt.Fprintf(w, "Avg %4vft %vmin\n", n/m, unit2min(m))

	fmt.Fprintf(w, "Deco(ft,min):\n")
	total := 0
	for i, h := range histo {
		if h == 0 || i == 0 {
			continue
		}
		min := unit2min(h)
		if min != 1 {
			fmt.Fprintf(w, "%4v %v\n", i, min-1)
			total += min - 1
		}
	}
	fmt.Fprintf(w, "total deco %v\n", total)
}

// FindMaxDepth returns the maximun depth reach on the dive
func (d *SWDive) FindMaxDepth() float64 {

	m := 0.0
	if d == nil {
		return m
	}

	for _, lr := range d.DiveLog.DiveLogRecords.DiveLogRecord {
		if lr.Depth > m {
			m = lr.Depth
		}
	}
	return m

}

// FindMaxDepth returns the depth with the closing time. Adjust1 is different than zero
// is added to each log entry time when comparing. The value is in hrs, positive or negative
// Common usage of it is 1
func (d *SWDive) FindBestMatch(target time.Time, adjust1 int) (float64, bool) {

	first := true
	depth := 0.0
	found := false

	var delta time.Duration

	for _, lr := range d.DiveLog.DiveLogRecords.DiveLogRecord {
		seconds := lr.Time
		if adjust1 != 0 {
			seconds += adjust1 * 60 * 60
		}
		t := d.DiveLog.startdate.Add(time.Duration(seconds) * time.Second)

		if target.Equal(t) {
			if debug {
				fmt.Fprintf(dwriter, "prefect match %v %v delta %v\n",
					target.Format(time.UnixDate),
					t.Format(time.UnixDate),
					delta)
			}
			return lr.Depth, found
		}

		if t.After(target) {
			if debug {
				fmt.Fprintf(dwriter, "past it  %v %v delta %v\n",
					target.Format(time.UnixDate),
					t.Format(time.UnixDate),
					delta)
			}
			// TODO: is this a better than prev match?
			return depth, found
		}

		if first {
			first = false
			found = true
			if target.Before(t) {
				// TODO: is this close enought to return?
				return lr.Depth, found
			}
			delta = target.Sub(t)
			depth = lr.Depth
			if debug {
				fmt.Fprintf(dwriter, "first %v %v delta %v\n",
					target.Format(time.UnixDate),
					t.Format(time.UnixDate),
					delta)
			}
			continue
		}

		if target.Sub(t) < delta {
			delta = target.Sub(t)
			depth = lr.Depth
			if debug {
				fmt.Fprintf(dwriter, "%v %v delta %v depth %v\n",
					target.Format(time.UnixDate),
					t.Format(time.UnixDate),
					delta, depth)
			}
		}
	}

	if found {
		if delta > time.Second*10 {
			return depth, false
		}
	}

	return depth, found
}

// PlayIt calculates the maximum and the minimun distance from the
// decompresion ceiling as it plays the log. It can also dump the tissues loading as it
// plays the log if pmodel is true
func (d *SWDive) PlayIt(m deco.Model,usePPO2 bool) (float64, float64) {
	//TODO: mix should come from the log
	logmix := deco.NewTrimix(18, 45)
	ccrmix := deco.NewTrimix(18, 45)
	if m == nil {
		m = deco.ZHL16C(0.20, 0.70)
	}

	time := 0.0
	lastminute := 0
	max := 0.0
	min := 10.0

	for _, lr := range d.DiveLog.DiveLogRecords.DiveLogRecord {
		depth := lr.Depth
		sloth := 10.0 / 60.0
		time += sloth
		
		if debug {
			fmt.Fprintf(dwriter,"playit: depth %v time %v\n",depth,time)
		}

		if usePPO2 {
			ccrmix = deco.CurrentCCRMix(logmix,deco.Feet2ATM(depth),lr.AveragePPO2)
		}

		m.LevelOff(sloth, depth, ccrmix)

		if minute := int(time); lastminute != minute {
			ceil := m.Ceiling()
			if debug {
				fmt.Fprintf(dwriter,"playit: Ceiling %v\n",ceil)
			}
			if pmodel {
				m.Print(true, fmt.Sprintf("playit%v", minute))
			}
			if depth > 10 {
				if debug {
					fmt.Fprintf(dwriter,"playit: depth %v ceil %v\n", depth, ceil)
				}
				if delta := depth - ceil; delta > max && minute > 60 && depth < 100 {
					max = delta
				}
				if delta := depth - ceil; delta < min && minute > 30 && depth < 100 {
					min = delta
					if debug {
						fmt.Fprintf(dwriter,"playit: ceiling distance reset at %v min / %2f ft  %2f\n",
							minute, depth, min)
					}
				}
			}
			lastminute = minute
		}
	}
	if debug {
		fmt.Fprintf(dwriter,"playit: max distance %f min distance %f\n", max, min)
	}
	return max, min
}
