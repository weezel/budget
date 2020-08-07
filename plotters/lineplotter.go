package plotters

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"io/ioutil"
	"time"
	"weezel/budget/external"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// chart "github.com/wcharczuk/go-chart"

const (
	spendingGraphHeight vg.Length = 4 * vg.Inch
	spendingGraphWidth  vg.Length = 6 * vg.Inch
)

func LineHistogramOfAnnualSpending(spending []external.SpendingHistory) ([]byte, error) {
	p, err := plot.New()
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: Plot initializing failed: %v", err)
		return []byte{}, errors.New(errMsg)
	}

	p.Title.Text = fmt.Sprintf("Kulutus vuonna %s", time.Now().Format("2006"))
	p.Y.Label.Text = "Kulutus per kuukausi"
	p.X.Label.Text = "Aika"
	p.X.Tick.Marker = plot.TimeTicks{Format: "01-2006"}

	var userAName string
	var userBName string

	userA := make(plotter.XYs, 0)
	userB := make(plotter.XYs, 0)
	for _, s := range spending {
		tmpPlotter := plotter.XY{}

		if userAName == "" {
			userAName = s.Username
		}
		if userBName == "" && s.Username != userAName {
			userBName = s.Username
		}

		if s.Username == userAName {
			tmpPlotter.X = float64(s.MonthYear.Unix())
			tmpPlotter.Y = float64(s.Spending)
			userA = append(userA, tmpPlotter)
		} else if s.Username == userBName {
			tmpPlotter.X = float64(s.MonthYear.Unix())
			tmpPlotter.Y = float64(s.Spending)
			userB = append(userB, tmpPlotter)
		}
	}

	lineUserA, pointsUserA, err := plotter.NewLinePoints(userA)
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: new line plots failed: %v", err)
		return []byte{}, errors.New(errMsg)
	}
	lineUserA.Color = color.RGBA{R: 132, G: 10, B: 219, A: 200}
	pointsUserA.Shape = draw.PlusGlyph{}
	pointsUserA.Color = color.RGBA{R: 162, G: 20, B: 219, A: 200}

	lineUserB, pointsUserB, err := plotter.NewLinePoints(userB)
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: new line plots failed: %v", err)
		return []byte{}, errors.New(errMsg)
	}
	lineUserB.Color = color.RGBA{R: 10, G: 100, B: 10, A: 155}
	pointsUserB.Shape = draw.PlusGlyph{}
	pointsUserB.Color = color.RGBA{R: 30, G: 110, B: 10, A: 155}
	p.Add(lineUserA, pointsUserA, lineUserB, pointsUserB, plotter.NewGrid())
	p.Legend.Add(userAName, lineUserA)
	p.Legend.Add(userBName, lineUserB)

	bin, err := p.WriterTo(spendingGraphWidth, spendingGraphHeight, "png")
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: writing spending histogram: %v", err)
		return []byte{}, errors.New(errMsg)
	}

	var imgTmp bytes.Buffer
	_, err = bin.WriteTo(&imgTmp)
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: 1 couldn't form histogram image file: %v", err)
		return []byte{}, errors.New(errMsg)
	}
	imgFile, err := ioutil.ReadAll(&imgTmp)
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: 2 couldn't form histogram image file: %v", err)
		return []byte{}, errors.New(errMsg)
	}

	return imgFile, nil
}
