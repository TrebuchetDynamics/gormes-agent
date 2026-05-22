package toon

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestEncodeJSON_GoldenFixtures(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "object with nested context and tabular array",
			json: `{
				"context":{"task":"Our favorite hikes together","location":"Boulder","season":"spring_2025"},
				"friends":["ana","luis","sam"],
				"hikes":[
					{"id":1,"name":"Blue Lake Trail","distanceKm":7.5,"elevationGain":320,"companion":"ana","wasSunny":true},
					{"id":2,"name":"Ridge Overlook","distanceKm":9.2,"elevationGain":540,"companion":"luis","wasSunny":false},
					{"id":3,"name":"Wildflower Loop","distanceKm":5.1,"elevationGain":180,"companion":"sam","wasSunny":true}
				]
			}`,
			want: "context:\n  task: Our favorite hikes together\n  location: Boulder\n  season: spring_2025\nfriends[3]: ana,luis,sam\nhikes[3]{id,name,distanceKm,elevationGain,companion,wasSunny}:\n  1,Blue Lake Trail,7.5,320,ana,true\n  2,Ridge Overlook,9.2,540,luis,false\n  3,Wildflower Loop,5.1,180,sam,true",
		},
		{
			name: "quotes ambiguous strings and keys",
			json: `{"order:id":7,"status":"true","note":"a:b","empty":"","dash":"-flag","text":"line1\nline2"}`,
			want: "\"order:id\": 7\nstatus: \"true\"\nnote: \"a:b\"\nempty: \"\"\ndash: \"-flag\"\ntext: \"line1\\nline2\"",
		},
		{
			name: "nonuniform objects use list items",
			json: `{"items":[{"id":1,"name":"First"},{"id":2,"name":"Second","extra":true}]}`,
			want: "items[2]:\n  - id: 1\n    name: First\n  - id: 2\n    name: Second\n    extra: true",
		},
		{
			name: "root primitive array",
			json: `["alpha","42",true,null]`,
			want: "[4]: alpha,\"42\",true,null",
		},
		{
			name: "empty arrays and objects",
			json: `{"items":[],"meta":{}}`,
			want: "items: []\nmeta:",
		},
		{
			name: "root empty array",
			json: `[]`,
			want: "[]",
		},
		{
			name: "nested empty array list item shorthand",
			json: `{"matrix":[[],["a","b"]]}`,
			want: "matrix[2]:\n  - []\n  - [2]: a,b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EncodeJSON([]byte(tt.json))
			if err != nil {
				t.Fatalf("EncodeJSON: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("EncodeJSON mismatch\nwant:\n%s\n\ngot:\n%s", tt.want, got)
			}
		})
	}
}

func TestDecodeJSON_GoldenFixtures(t *testing.T) {
	tests := []struct {
		name string
		toon string
		want string
	}{
		{
			name: "tabular array",
			toon: "items[2]{sku,qty,price}:\n  A1,2,9.99\n  B2,1,14.5",
			want: `{"items":[{"sku":"A1","qty":2,"price":9.99},{"sku":"B2","qty":1,"price":14.5}]}`,
		},
		{
			name: "expanded object list",
			toon: "items[2]:\n  - id: 1\n    name: First\n  - id: 2\n    name: Second\n    extra: true",
			want: `{"items":[{"id":1,"name":"First"},{"id":2,"name":"Second","extra":true}]}`,
		},
		{
			name: "nested object and primitive array",
			toon: "context:\n  task: Our favorite hikes together\n  location: Boulder\nfriends[3]: ana,luis,sam",
			want: `{"context":{"task":"Our favorite hikes together","location":"Boulder"},"friends":["ana","luis","sam"]}`,
		},
		{
			name: "quoted values remain strings",
			toon: "status: \"true\"\ncount: \"42\"\nlead: \"05\"\nempty: \"\"",
			want: `{"status":"true","count":"42","lead":"05","empty":""}`,
		},
		{
			name: "quoted unicode surrogate pair escapes",
			toon: `emoji: "faces \ud83d\ude00"`,
			want: `{"emoji":"faces 😀"}`,
		},
		{
			name: "quoted JSON-compatible escapes",
			toon: `path: "https:\/\/example.com\/a"
backspace: "a\bb"
formfeed: "a\fb"`,
			want: `{"path":"https://example.com/a","backspace":"a\bb","formfeed":"a\fb"}`,
		},
		{
			name: "nested empty array list item shorthand",
			toon: "matrix[2]:\n  - []\n  - [2]: a,b",
			want: `{"matrix":[[],["a","b"]]}`,
		},
		{
			name: "root empty array with trailing spaces",
			toon: "[]   \n",
			want: `[]`,
		},
		{
			name: "root empty object shorthand with trailing spaces",
			toon: "{}   \n",
			want: `{}`,
		},
		{
			name: "nested empty object shorthand",
			toon: "meta: {}\nitems[2]:\n  - {}\n  - name: Ada",
			want: `{"meta":{},"items":[{},{"name":"Ada"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeJSON([]byte(tt.toon))
			if err != nil {
				t.Fatalf("DecodeJSON: %v", err)
			}
			assertJSONEqual(t, tt.want, string(got))
		})
	}
}

func TestEncodeDecodeJSON_RoundTrip(t *testing.T) {
	tests := []string{
		`{"context":{"task":"summarize","priority":2},"rows":[{"id":1,"ok":true},{"id":2,"ok":false}]}`,
		`{"items":[{"id":1,"tags":["a","b"]},{"id":2,"tags":[]}]}`,
		`[{"id":1,"name":"Ada"},{"id":2,"name":"Bob"}]`,
		`{"number":1000000,"decimal":0.000001,"text":"hello, world","nil":null}`,
	}

	for _, input := range tests {
		encoded, err := EncodeJSON([]byte(input))
		if err != nil {
			t.Fatalf("EncodeJSON(%s): %v", input, err)
		}
		decoded, err := DecodeJSON(encoded)
		if err != nil {
			t.Fatalf("DecodeJSON(%s): %v", encoded, err)
		}
		assertJSONEqual(t, input, string(decoded))
	}
}

func TestDecodeJSON_StrictTabularCount(t *testing.T) {
	_, err := DecodeJSON([]byte("items[2]{id,name}:\n  1,Ada"))
	if err == nil {
		t.Fatal("DecodeJSON succeeded; want strict row-count error")
	}
}

func BenchmarkEncodeJSON_TOON(b *testing.B) {
	raw := []byte(`{"rows":[{"id":1,"name":"Ada","role":"admin","active":true},{"id":2,"name":"Bob","role":"user","active":false},{"id":3,"name":"Cam","role":"ops","active":true}],"meta":{"source":"bench","count":3}}`)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		out, err := EncodeJSON(raw)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(out)), "toon_bytes/op")
		b.ReportMetric(float64(estimatedTokens(out)), "toon_est_tokens/op")
	}
}

func BenchmarkEncodeJSON_EncodingJSONCompact(b *testing.B) {
	raw := []byte(`{"rows":[{"id":1,"name":"Ada","role":"admin","active":true},{"id":2,"name":"Bob","role":"user","active":false},{"id":3,"name":"Cam","role":"ops","active":true}],"meta":{"source":"bench","count":3}}`)
	b.ReportAllocs()
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := json.Compact(&buf, raw); err != nil {
			b.Fatal(err)
		}
		out := buf.Bytes()
		b.ReportMetric(float64(len(out)), "json_bytes/op")
		b.ReportMetric(float64(estimatedTokens(out)), "json_est_tokens/op")
	}
}

func assertJSONEqual(t *testing.T, want, got string) {
	t.Helper()
	var wantValue any
	var gotValue any
	wantDec := json.NewDecoder(bytes.NewReader([]byte(want)))
	wantDec.UseNumber()
	if err := wantDec.Decode(&wantValue); err != nil {
		t.Fatalf("bad expected JSON: %v", err)
	}
	gotDec := json.NewDecoder(bytes.NewReader([]byte(got)))
	gotDec.UseNumber()
	if err := gotDec.Decode(&gotValue); err != nil {
		t.Fatalf("bad actual JSON: %v\n%s", err, got)
	}
	wantCanonical, _ := json.Marshal(wantValue)
	gotCanonical, _ := json.Marshal(gotValue)
	if !bytes.Equal(wantCanonical, gotCanonical) {
		t.Fatalf("JSON mismatch\nwant: %s\ngot:  %s", wantCanonical, gotCanonical)
	}
}

func estimatedTokens(raw []byte) int {
	if len(raw) == 0 {
		return 0
	}
	return (len(raw) + 3) / 4
}
