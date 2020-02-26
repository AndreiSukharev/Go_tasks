package main

import (
	"bufio"
	"bytes"
	json "encoding/json"
	"fmt"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
	"io"
	"os"
	"regexp"
	"strings"
)

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	defer file.Close()
	if err != nil {
		panic(err)
	}

	r := regexp.MustCompile("@")
	seenBrowsers := make(map[string]bool)

	uniqueBrowsers := 0
	foundUsers := make([]string, 0, 115)
	reader := bufio.NewReader(file)
	i := -1
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		i++
		user := Browser{}
		err = user.UnmarshalJSON(line)
		if err != nil {
			panic(err)
		}

		isAndroid := false
		isMSIE := false
		browsers := user.Browsers

		for _, browser := range browsers {
			if ok := strings.Contains(browser, "Android"); ok {
				isAndroid = true
				notSeenBefore := true

				if exist := seenBrowsers[browser]; exist {
						notSeenBefore = false
				}

				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers[browser] = true
					uniqueBrowsers++
				}
			}

			if ok := strings.Contains(browser, "MSIE"); ok {
				isMSIE = true
				notSeenBefore := true
				if exist := seenBrowsers[browser]; exist {
					notSeenBefore = false
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers[browser] = true
					uniqueBrowsers++
				}
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}
		email := r.ReplaceAllString(user.Email, " [at] ")
		foundUsers = append(foundUsers, fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email))

	}

	fmt.Fprintln(out, "found users:\n"+ strings.Join(foundUsers, ""))
	fmt.Fprintln(out, "Total unique browsers", uniqueBrowsers)

}

func main() {
	fastOut := new(bytes.Buffer)
	FastSearch(fastOut)
	fastResult := fastOut.String()
	fmt.Println(fastResult)
}

type Browser struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:company`
	Country  string   `json:country`
	Email    string   `json:"email"`
	Job      string   `json:job`
	Name     string   `json:"name"`
	Phone    string   `json:phone`
}


var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjson9f2eff5fDecodeMystruct(in *jlexer.Lexer, out *Browser) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "browsers":
			if in.IsNull() {
				in.Skip()
				out.Browsers = nil
			} else {
				in.Delim('[')
				if out.Browsers == nil {
					if !in.IsDelim(']') {
						out.Browsers = make([]string, 0, 4)
					} else {
						out.Browsers = []string{}
					}
				} else {
					out.Browsers = (out.Browsers)[:0]
				}
				for !in.IsDelim(']') {
					var v1 string
					v1 = string(in.String())
					out.Browsers = append(out.Browsers, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "Company":
			out.Company = string(in.String())
		case "Country":
			out.Country = string(in.String())
		case "email":
			out.Email = string(in.String())
		case "Job":
			out.Job = string(in.String())
		case "name":
			out.Name = string(in.String())
		case "Phone":
			out.Phone = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson9f2eff5fEncodeMystruct(out *jwriter.Writer, in Browser) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"browsers\":"
		out.RawString(prefix[1:])
		if in.Browsers == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v2, v3 := range in.Browsers {
				if v2 > 0 {
					out.RawByte(',')
				}
				out.String(string(v3))
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"Company\":"
		out.RawString(prefix)
		out.String(string(in.Company))
	}
	{
		const prefix string = ",\"Country\":"
		out.RawString(prefix)
		out.String(string(in.Country))
	}
	{
		const prefix string = ",\"email\":"
		out.RawString(prefix)
		out.String(string(in.Email))
	}
	{
		const prefix string = ",\"Job\":"
		out.RawString(prefix)
		out.String(string(in.Job))
	}
	{
		const prefix string = ",\"name\":"
		out.RawString(prefix)
		out.String(string(in.Name))
	}
	{
		const prefix string = ",\"Phone\":"
		out.RawString(prefix)
		out.String(string(in.Phone))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v Browser) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson9f2eff5fEncodeMystruct(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v Browser) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson9f2eff5fEncodeMystruct(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *Browser) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson9f2eff5fDecodeMystruct(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *Browser) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson9f2eff5fDecodeMystruct(l, v)
}