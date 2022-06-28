package svg

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
)

func Example() {
	gbr := `G04 Ucamco ex. 1: Two square boxes*
%MOMM*%
%FSLAX26Y26*%
%LPD*%
%ADD10C,0.010*%
D10*
X0Y0D02*
G01*
X5000000Y0D01*
Y5000000D01*
X0D01*
Y0D01*
X6000000D02*
X11000000D01*
Y5000000D01*
X6000000D01*
Y0D01*
M02*`

	svgP, err := SVG(bytes.NewBufferString(gbr))
	if err != nil {
		log.Fatalf("%+v", err)
	}
	svgP.PanZoom = false
	buf := bytes.NewBuffer(nil)
	if err := svgP.Write(buf); err != nil {
		log.Fatalf("%+v", err)
	}
	fmt.Printf("%s", buf.Bytes())

	// Output:
	// <svg viewBox="0 -5000000 11000000 5000000" style="background-color: black;" xmlns="http://www.w3.org/2000/svg">
	// <line x1="0" y1="-0" x2="5000000" y2="-0" stroke-width="10000" stroke-linecap="round" stroke="white" line="8"/>
	// <line x1="5000000" y1="-0" x2="5000000" y2="-5000000" stroke-width="10000" stroke-linecap="round" stroke="white" line="9"/>
	// <line x1="5000000" y1="-5000000" x2="0" y2="-5000000" stroke-width="10000" stroke-linecap="round" stroke="white" line="10"/>
	// <line x1="0" y1="-5000000" x2="0" y2="-0" stroke-width="10000" stroke-linecap="round" stroke="white" line="11"/>
	// <line x1="6000000" y1="-0" x2="11000000" y2="-0" stroke-width="10000" stroke-linecap="round" stroke="white" line="13"/>
	// <line x1="11000000" y1="-0" x2="11000000" y2="-5000000" stroke-width="10000" stroke-linecap="round" stroke="white" line="14"/>
	// <line x1="11000000" y1="-5000000" x2="6000000" y2="-5000000" stroke-width="10000" stroke-linecap="round" stroke="white" line="15"/>
	// <line x1="6000000" y1="-5000000" x2="6000000" y2="-0" stroke-width="10000" stroke-linecap="round" stroke="white" line="16"/>
	// </svg>
}

func TestGerber(t *testing.T) {
	outDir, err := os.MkdirTemp("", t.Name())
	if err != nil {
		t.Fatalf("%+v", err)
	}
	// t.Logf("outDir %s", outDir)
	defer os.RemoveAll(outDir)

	src := filepath.Join("testdata", "Gerber", "clockblock-F_Cu.gbr")
	dst := filepath.Join(outDir, "dst.svg")

	err = func() error {
		srcF, err := os.Open(src)
		if err != nil {
			return errors.Wrap(err, "")
		}
		defer srcF.Close()
		svgP, err := SVG(srcF)
		if err != nil {
			return errors.Wrap(err, "")
		}
		svgP.PolarityDark = "white"
		svgP.PolarityClear = "black"

		dstF, err := os.Create(dst)
		if err != nil {
			return errors.Wrap(err, "")
		}
		defer dstF.Close()
		if err := svgP.Write(dstF); err != nil {
			return errors.Wrap(err, "")
		}
		return nil
	}()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	expected := filepath.Join("testdata", "Gerber", "clockblock-F_Cu.svg")
	if err := diffFiles(dst, expected); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestProcessorJSONMarshal(t *testing.T) {
	p := Processor{}
	p.Data = append(p.Data, Circle{X: 11, Y: 23})
	svgpath := Path{}
	svgpath.Commands = append(svgpath.Commands, PathLine{X: 31, Y: 63})
	svgpath.Commands = append(svgpath.Commands, PathArc{CenterX: -27, CenterY: -87})
	p.Data = append(p.Data, svgpath)

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	unmarshaled := Processor{}
	if err := json.Unmarshal(b, &unmarshaled); err != nil {
		t.Fatalf("%+v", err)
	}

	e0, ok := unmarshaled.Data[0].(Circle)
	if !ok {
		t.Fatalf("%+v", unmarshaled)
	}
	if e0.X != 11 || e0.Y != 23 {
		t.Fatalf("%+v", unmarshaled)
	}

	e1, ok := unmarshaled.Data[1].(Path)
	if !ok {
		t.Fatalf("%+v", unmarshaled)
	}
	p0, ok := e1.Commands[0].(PathLine)
	if !ok {
		t.Fatalf("%+v", unmarshaled)
	}
	if p0.X != 31 || p0.Y != 63 {
		t.Fatalf("%+v", unmarshaled)
	}
	p1, ok := e1.Commands[1].(PathArc)
	if !ok {
		t.Fatalf("%+v", unmarshaled)
	}
	if p1.CenterX != -27 || p1.CenterY != -87 {
		t.Fatalf("%+v", unmarshaled)
	}
}

func diff(ar, br io.Reader) error {
	ab, bb := bufio.NewReader(ar), bufio.NewReader(br)
	for i := 0; ; i++ {
		a, aerr := ab.ReadByte()
		b, berr := bb.ReadByte()
		if aerr != nil || berr != nil {
			if aerr == io.EOF && berr == io.EOF {
				return nil
			}
			return errors.Errorf(`"%+v" "%+v"`, aerr, berr)
		}
		if a != b {
			return errors.Errorf("%d %v %v", i, a, b)
		}
	}
}

func diffFiles(a, b string) error {
	aF, err := os.Open(a)
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer aF.Close()

	bF, err := os.Open(b)
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer bF.Close()

	if err := diff(aF, bF); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}
