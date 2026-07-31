package source

import (
	"testing"
)

func TestSourceConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     SourceConfig
		wantErr bool
	}{
		{
			name: "valid minimal config",
			cfg: SourceConfig{
				ID:      "testsource",
				Name:    "Test Source",
				BaseURL: "https://example.com",
				Enabled: true,
				Search: SearchConfig{
					Path:       "/",
					Method:     "GET",
					QueryParam: "q",
					Results: ResultsConfig{
						Selector: ".result-row",
						Title:    FieldDef{Selector: "h3 a"},
						Link:     FieldDef{Selector: "h3 a", Attribute: "href"},
					},
				},
				Download: DownloadConfig{
					PostPage: PostPageConfig{
						Button: ButtonConfig{Selector: ".download-btn", Attribute: "href"},
					},
					Method: "GET",
				},
			},
			wantErr: false,
		},
		{
			name: "valid with empty title and link selectors (using row itself)",
			cfg: SourceConfig{
				ID:      "rowbased",
				Name:    "Row Based",
				BaseURL: "https://example.com",
				Enabled: true,
				Search: SearchConfig{
					Path:       "/",
					Method:     "GET",
					QueryParam: "s",
					Results: ResultsConfig{
						Selector: "a.result-link",
						Title:    FieldDef{Selector: ""},
						Link:     FieldDef{Selector: "", Attribute: "href"},
					},
				},
				Download: DownloadConfig{
					PostPage: PostPageConfig{
						Button: ButtonConfig{Selector: ".btn", Attribute: "href"},
					},
					Method: "GET",
				},
			},
			wantErr: false,
		},
		{
			name: "valid with filename header and POST method",
			cfg: SourceConfig{
				ID:      "baiscopelk",
				Name:    "BaiscopeLK",
				BaseURL: "https://www.baiscopelk.com",
				Enabled: true,
				Search: SearchConfig{
					Path:       "/",
					Method:     "GET",
					QueryParam: "s",
					Results: ResultsConfig{
						Selector: "article.elementor-post",
						Title:    FieldDef{Selector: "h5.post-title a"},
						Link:     FieldDef{Selector: "a.link", Attribute: "href"},
					},
				},
				Download: DownloadConfig{
					PostPage: PostPageConfig{
						Button: ButtonConfig{Selector: "a.dl-btn", Attribute: "href"},
					},
					Method:         "POST",
					Headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
					FilenameHeader: "X-Dlm-File-Name",
				},
			},
			wantErr: false,
		},
		{
			name: "missing id",
			cfg: SourceConfig{
				ID:      "",
				Name:    "No ID",
				BaseURL: "https://example.com",
				Search: SearchConfig{
					Results: ResultsConfig{
						Selector: ".row",
						Title:    FieldDef{Selector: "a"},
						Link:     FieldDef{Selector: "a", Attribute: "href"},
					},
				},
				Download: DownloadConfig{
					PostPage: PostPageConfig{
						Button: ButtonConfig{Selector: ".btn", Attribute: "href"},
					},
					Method: "GET",
				},
			},
			wantErr: true,
		},
		{
			name: "id with invalid characters",
			cfg: SourceConfig{
				ID:      "Bad ID!",
				Name:    "Bad ID",
				BaseURL: "https://example.com",
				Search: SearchConfig{
					Results: ResultsConfig{
						Selector: ".row",
						Title:    FieldDef{Selector: "a"},
						Link:     FieldDef{Selector: "a", Attribute: "href"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			cfg: SourceConfig{
				ID:      "noname",
				Name:    "",
				BaseURL: "https://example.com",
				Search: SearchConfig{
					Path:       "/",
					Method:     "GET",
					QueryParam: "q",
					Results: ResultsConfig{
						Selector: ".row",
						Title:    FieldDef{Selector: "a"},
						Link:     FieldDef{Selector: "a", Attribute: "href"},
					},
				},
				Download: DownloadConfig{
					PostPage: PostPageConfig{
						Button: ButtonConfig{Selector: ".btn", Attribute: "href"},
					},
					Method: "GET",
				},
			},
			wantErr: true,
		},
		{
			name: "missing baseURL",
			cfg: SourceConfig{
				ID:      "nobase",
				Name:    "No Base",
				BaseURL: "",
				Search: SearchConfig{
					Results: ResultsConfig{
						Selector: ".row",
						Title:    FieldDef{Selector: "a"},
						Link:     FieldDef{Selector: "a", Attribute: "href"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid baseURL scheme",
			cfg: SourceConfig{
				ID:      "badurl",
				Name:    "Bad URL",
				BaseURL: "ftp://example.com",
				Search: SearchConfig{
					Results: ResultsConfig{
						Selector: ".row",
						Title:    FieldDef{Selector: "a"},
						Link:     FieldDef{Selector: "a", Attribute: "href"},
					},
				},
				Download: DownloadConfig{
					PostPage: PostPageConfig{
						Button: ButtonConfig{Selector: ".btn", Attribute: "href"},
					},
					Method: "GET",
				},
			},
			wantErr: true,
		},
		{
			name: "missing results selector",
			cfg: SourceConfig{
				ID:      "nosel",
				Name:    "No Selector",
				BaseURL: "https://example.com",
				Search: SearchConfig{
					Results: ResultsConfig{
						Title: FieldDef{Selector: "a"},
						Link:  FieldDef{Selector: "a", Attribute: "href"},
					},
				},
				Download: DownloadConfig{
					PostPage: PostPageConfig{
						Button: ButtonConfig{Selector: ".btn", Attribute: "href"},
					},
					Method: "GET",
				},
			},
			wantErr: true,
		},
		{
			name: "missing download button selector",
			cfg: SourceConfig{
				ID:      "nodlbtn",
				Name:    "No Download Button",
				BaseURL: "https://example.com",
				Search: SearchConfig{
					Path:       "/",
					Method:     "GET",
					QueryParam: "q",
					Results: ResultsConfig{
						Selector: ".row",
						Title:    FieldDef{Selector: "a"},
						Link:     FieldDef{Selector: "a", Attribute: "href"},
					},
				},
				Download: DownloadConfig{
					PostPage: PostPageConfig{},
					Method:   "GET",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid download method",
			cfg: SourceConfig{
				ID:      "baddl",
				Name:    "Bad DL Method",
				BaseURL: "https://example.com",
				Search: SearchConfig{
					Path:       "/",
					Method:     "GET",
					QueryParam: "q",
					Results: ResultsConfig{
						Selector: ".row",
						Title:    FieldDef{Selector: "a"},
						Link:     FieldDef{Selector: "a", Attribute: "href"},
					},
				},
				Download: DownloadConfig{
					PostPage: PostPageConfig{
						Button: ButtonConfig{Selector: ".btn", Attribute: "href"},
					},
					Method: "DELETE",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
