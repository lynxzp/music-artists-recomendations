package lastfm

import (
	"testing"
)

func TestAppendSimilarArtists(t *testing.T) {
	tests := []struct {
		name   string
		a      []SimilarArtist
		b      []SimilarArtist
		weight float64
		want   map[string]float64 // expected normalized matches
	}{
		{
			name: "no duplicates weight 1",
			a: []SimilarArtist{
				{Name: "Artist1", Match: 100},
				{Name: "Artist2", Match: 50},
			},
			b: []SimilarArtist{
				{Name: "Artist3", Match: 80},
				{Name: "Artist4", Match: 40},
			},
			weight: 1.0,
			want: map[string]float64{
				"Artist1": 100,
				"Artist2": 50,
				"Artist3": 80,
				"Artist4": 40,
			},
		},
		{
			name: "with duplicates sums matches",
			a: []SimilarArtist{
				{Name: "Artist1", Match: 50},
				{Name: "Artist2", Match: 50},
			},
			b: []SimilarArtist{
				{Name: "Artist2", Match: 50},
				{Name: "Artist3", Match: 25},
			},
			weight: 1.0,
			want: map[string]float64{
				"Artist1": 50,
				"Artist2": 100, // 50 + 50
				"Artist3": 25,
			},
		},
		{
			name: "weight scales b values",
			a: []SimilarArtist{
				{Name: "Artist1", Match: 100},
			},
			b: []SimilarArtist{
				{Name: "Artist2", Match: 100},
			},
			weight: 0.5,
			want: map[string]float64{
				"Artist1": 100, // max
				"Artist2": 50,  // 100 * 0.5
			},
		},
		{
			name:   "empty b returns a unchanged",
			a: []SimilarArtist{
				{Name: "Artist1", Match: 80},
				{Name: "Artist2", Match: 40},
			},
			b:      []SimilarArtist{},
			weight: 1.0,
			want: map[string]float64{
				"Artist1": 80,
				"Artist2": 40,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendSimilarArtists(tt.a, tt.b, tt.weight)

			if len(got) != len(tt.want) {
				t.Errorf("got %d artists, want %d", len(got), len(tt.want))
				return
			}

			for _, artist := range got {
				expected, ok := tt.want[artist.Name]
				if !ok {
					t.Errorf("unexpected artist %q in result", artist.Name)
					continue
				}
				if artist.Match != expected {
					t.Errorf("artist %q: got match %.2f, want %.2f", artist.Name, artist.Match, expected)
				}
			}
		})
	}
}