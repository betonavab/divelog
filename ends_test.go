package divelog

import (
	"io/ioutil"
	"testing"
	"time"
)

func Test_NewDive(t *testing.T) {
	logfile := "sw.xml"
	_, err := NewShearwaterLog(logfile)
	if err != nil {
		t.Errorf("can't open %v", logfile)
	}
}

func Test_FindMaxDepth(t *testing.T) {
	logfile := "sw.xml"
	l, err := NewShearwaterLog(logfile)
	if err != nil {
		t.Errorf("can't open %v", logfile)
	}
	max := l.FindMaxDepth()
	if max != 161.4 {
		t.Errorf("FindMaxDepth is %v; want some %v", max, 161.4)
	}
}

func Test_FindBestMatch(t *testing.T) {
	logfile := "sw.xml"
	l, err := NewShearwaterLog(logfile)
	if err != nil {
		t.Errorf("can't open %v", logfile)
		return
	}
	date := "2019:11:07 14:45:32"
	target, err := time.Parse("2006:01:02 15:04:05", date)
	if err != nil {
		t.Errorf("can't parse %v", logfile)
		return
	}
	depth, found := l.FindBestMatch(target, 0)
	if !found {
		t.Errorf("%v depth was %v \n", target.Format(time.UnixDate), depth)
		return
	}

	if depth != 160.6 {
		t.Errorf("FindBestMatch is %v; want some %v", depth, 160.6)
		return
	}
}

func Test_Print(t *testing.T) {
	logfile := "sw.xml"
	l, err := NewShearwaterLog(logfile)
	if err != nil {
		t.Errorf("can't open %v", logfile)
		return
	}

	l.PrintAll(ioutil.Discard)
	l.PrintHisto(ioutil.Discard)
}


func Test_interface(t *testing.T) {
	logfile := "sw.xml"
	l, err := NewShearwaterLog(logfile)
	if err != nil {
		t.Errorf("can't open %v", logfile)
		return
	}

	l.PrintAll(ioutil.Discard)
	l.PrintHisto(ioutil.Discard)

	max := l.FindMaxDepth()
	if max != 161.4 {
		t.Errorf("FindMaxDepth is %v; want some %v", max, 161.4)
	}
}

func Test_Playit(t *testing.T) {
	logfile := "sw.xml"
	l, err := NewShearwaterLog(logfile)
	if err != nil {
		t.Errorf("can't open %v", logfile)
		return
	}

	max,min := l.PlayIt(nil,true)

	if max != 11.900000000000006 {
		t.Errorf("ReportCeilingDistance(max) is %v; want some %v", 
				max, 11.9)

	}
	if min != -4 {
		t.Errorf("ReportCeilingDistance(mix) is %v; want some %v", 
				min, -4)

	}
}

func Test_debug(t *testing.T) {
	EnableDebug(ioutil.Discard)
	DisableDebug()

	EnablePmodel(ioutil.Discard)
	DisablePmodel()

}
