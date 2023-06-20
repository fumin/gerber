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
	"reflect"
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
	p := Processor{
		MinX:          123,
		MaxX:          321,
		MinY:          111,
		MaxY:          222,
		Decimal:       1.2,
		PolarityDark:  "dark-color",
		PolarityClear: "clear-color",
		Scale:         1.3,
		Width:         "ww",
		Height:        "hh",
		PanZoom:       true,
	}
	p.Data = append(p.Data, Circle{Type: ElementTypeCircle, Line: 33, X: 11, Y: 23, Radius: 55, Fill: "circle-fill"})
	p.Data = append(p.Data, Rectangle{Type: ElementTypeRectangle, Line: 31, Aperture: "rect-aper", X: 23, Y: 24, Width: 33, Height: 44, RX: 87, RY: 98, Fill: "rect-fill"})
	svgpath := Path{Type: ElementTypePath, Line: 2000, X: 2001, Y: 2002, Fill: "path-fill"}
	svgpath.Commands = append(svgpath.Commands, PathLine{Type: ElementTypeLine, X: 31, Y: 63})
	svgpath.Commands = append(svgpath.Commands, PathArc{Type: ElementTypeArc, RadiusX: -11, RadiusY: -12, LargeArc: 3, Sweep: 4, X: 57, Y: 58, CenterX: -27, CenterY: -87})
	p.Data = append(p.Data, svgpath)
	p.Data = append(p.Data, Line{Type: ElementTypeLine, Line: 1111, X1: 2222, Y1: 3333, X2: 4444, Y2: 5555, StrokeWidth: 6666, Cap: "line-cap", Stroke: "line-stroke"})
	p.Data = append(p.Data, Arc{Type: ElementTypeArc, Line: -1111, XS: -2222, YS: -3333, RadiusX: -4444, RadiusY: -5555, LargeArc: -6666, Sweep: -7777, XE: -8888, YE: -9999, StrokeWidth: -1234, CenterX: -1235, CenterY: -1236, Stroke: "arc-stroke"})

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	unmarshaled := Processor{}
	if err := json.Unmarshal(b, &unmarshaled); err != nil {
		t.Fatalf("%+v", err)
	}

	if !reflect.DeepEqual(unmarshaled, p) {
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
