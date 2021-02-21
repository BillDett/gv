package main

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"testing"
)

//go:embed gulliver.txt
var bigtext string

//go:embed asyoulik.txt
var mediumtext string

func TestPieceTable(t *testing.T) {

	base := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	var pt *PieceTable
	var result, answer string

	fmt.Println("Insert at beginning of a piece")
	pt = NewPieceTable(base)
	pt.Insert(0, "FOO")
	result = pt.Text()
	answer = "FOO" + base
	if result != answer {
		t.Errorf("Fail: Beginning insert wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Insert at end of a piece")
	pt = NewPieceTable(base)
	pt.Insert(26, "FOO")
	result = pt.Text()
	answer = base + "FOO"
	if result != answer {
		t.Errorf("Fail: End insert wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Insert in middle of a piece")
	pt = NewPieceTable(base)
	pt.Insert(13, "FOO")
	result = pt.Text()
	answer = base[:13] + "FOO" + base[13:]
	if result != answer {
		t.Errorf("Fail: Middle insert wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Multiple inserts")
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

	fmt.Println("Delete across multiple pieces")
	pt.Delete(6, 9)
	result = pt.Text()
	answer = base[:6] + "R" + base[7:9] + "123" + base[9:11] + "456789" + base[11:]
	if result != answer {
		t.Errorf("Fail: Middle insert  wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Delete at beginning of a piece")
	pt = NewPieceTable(base)
	pt.Delete(0, 5)
	//pt.Dump()
	result = pt.Text()
	answer = base[5:]
	if result != answer {
		t.Errorf("Fail: Beginning delete wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Delete at end of a piece")
	pt = NewPieceTable(base)
	pt.Delete(len(base)-5, 5)
	result = pt.Text()
	answer = base[:len(base)-5]
	if result != answer {
		t.Errorf("Fail: End delete wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Delete in middle of a piece")
	pt = NewPieceTable(base)
	pt.Delete(13, 6)
	result = pt.Text()
	answer = base[:13] + base[19:]
	if result != answer {
		t.Errorf("Fail: Middle delete wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Add to Empty PieceTable")
	pt = NewPieceTable("")
	answer = "The quick brown fox jumped over the small dog."
	for i := 0; i < len(answer); i++ {
		pt.Insert(i, string(answer[i]))
	}
	result = pt.Text()
	if result != answer {
		t.Errorf("Fail: Append to empty, wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Create a large PieceTable from a huge string")
	pt = NewPieceTable("")
	pt.Insert(0, bigtext)
	result = pt.Text()
	if result != bigtext {
		t.Errorf("Fail: Create into empty bigtext, wanted >%s< got >%s<\n", answer, result)
	}

	fmt.Println("Create a large PieceTable by appending chunks of a huge string")
	pt = NewPieceTable("")
	chunk := 25
	answer = mediumtext
	//answer = bigtext
	//answer = base
	for i := 0; i < len(answer); i += chunk {
		if i+chunk < len(answer) {
			pt.Append(answer[i : i+chunk])
		} else {
			pt.Append(answer[i:])
		}
	}
	result = pt.Text()
	if result != answer {
		err := ioutil.WriteFile("largetext_answer.txt", []byte(answer), 0644)
		if err != nil {
			t.Error("Fail: Append test couldn't write to answer file")
		}
		err = ioutil.WriteFile("largetext_result.txt", []byte(result), 0644)
		if err != nil {
			t.Error("Fail: Append test couldn't write to result file")
		}

		t.Errorf("Fail: Append to empty bigtext failed, see files.\n")
	}

}
