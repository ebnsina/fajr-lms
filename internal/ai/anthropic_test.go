package ai

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		text string
		want int
		bad  bool
	}{
		{name: "plain json", text: `{"questions":[{"kind":"true_false","prompt":"Arabic reads right to left.","points":1,"options":[{"label":"True","is_correct":true},{"label":"False"}]}]}`, want: 1},
		{name: "wrapped in a fence", text: "here you go\n```json\n{\"questions\":[{\"kind\":\"mcq_single\",\"prompt\":\"How many?\",\"points\":2,\"options\":[{\"label\":\"Two\"},{\"label\":\"Three\",\"is_correct\":true},{\"label\":\"Four\"}]}]}\n```", want: 1},
		{name: "prose instead of json", text: "I would rather explain the lesson.", bad: true},
		{name: "an empty list", text: `{"questions":[]}`, bad: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parse(c.text)
			if c.bad {
				if err == nil {
					t.Fatalf("got %d questions, want an error", len(got))
				}
				return
			}
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(got) != c.want {
				t.Fatalf("got %d questions, want %d", len(got), c.want)
			}
		})
	}
}
