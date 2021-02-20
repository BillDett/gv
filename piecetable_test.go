package main

import (
	"testing"
)

func TestPieceTable(t *testing.T) {

	base := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	var pt *PieceTable
	var result, answer string

	// Insert at beginning of a piece
	pt = NewPieceTable(base)
	pt.Insert(0, "FOO")
	result = pt.Text()
	answer = "FOO" + base
	if result != answer {
		t.Errorf("Fail: Beginning insert wanted >%s< got >%s<\n", answer, result)
	}

	// Insert at end of a piece
	pt = NewPieceTable(base)
	pt.Insert(26, "FOO")
	result = pt.Text()
	answer = base + "FOO"
	if result != answer {
		t.Errorf("Fail: End insert wanted >%s< got >%s<\n", answer, result)
	}

	// Insert in middle of a piece
	pt = NewPieceTable(base)
	pt.Insert(13, "FOO")
	result = pt.Text()
	answer = base[:13] + "FOO" + base[13:]
	if result != answer {
		t.Errorf("Fail: Middle insert wanted >%s< got >%s<\n", answer, result)
	}

	// Multiple inserts
	pt = NewPieceTable(base)
	pt.Insert(5, "FOO")
	pt.Insert(10, "BAR")
	pt.Insert(15, "123")
	pt.Insert(20, "456789")
	pt.Insert(7, "abc")
	result = pt.Text()
	answer = base[:5] + "FOabcO" + base[5:7] + "BAR" + base[7:9] + "123" + base[9:11] + "456789" + base[11:]
	if result != answer {
		t.Errorf("Fail: Middle insert  wanted >%s< got >%s<\n", answer, result)
	}

	// Delete across multiple pieces
	pt.Delete(6, 9)
	result = pt.Text()
	answer = base[:6] + "R" + base[7:9] + "123" + base[9:11] + "456789" + base[11:]
	if result != answer {
		t.Errorf("Fail: Middle insert  wanted >%s< got >%s<\n", answer, result)
	}

	// Delete at beginning of a piece
	pt = NewPieceTable(base)
	pt.Delete(0, 5)
	//pt.Dump()
	result = pt.Text()
	answer = base[5:]
	if result != answer {
		t.Errorf("Fail: Beginning delete wanted >%s< got >%s<\n", answer, result)
	}

	// Delete at end of a piece
	pt = NewPieceTable(base)
	pt.Delete(len(base)-5, 5)
	result = pt.Text()
	answer = base[:len(base)-5]
	if result != answer {
		t.Errorf("Fail: End insert  wanted >%s< got >%s<\n", answer, result)
	}

}
