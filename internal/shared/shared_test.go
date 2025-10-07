package shared

import "testing"

func TestNormalizeTrackKey(t *testing.T) {
	tc := []struct {
		name   string
		title  string
		artist string
		want   string
	}{
		{
			name:   "basic normalization",
			title:  "Song Title",
			artist: "Artist Name",
			want:   "song title|artist name",
		},
		{
			name:   "extra whitespace",
			title:  "  Song   Title  ",
			artist: "  Artist   Name  ",
			want:   "song title|artist name",
		},
		{
			name:   "mixed case",
			title:  "SoNg TiTlE",
			artist: "ArTiSt NaMe",
			want:   "song title|artist name",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTrackKey(tt.title, tt.artist)
			if got != tt.want {
				t.Errorf("normalizeTrackKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
