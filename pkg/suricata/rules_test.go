package suricata

import (
   "net/http/httptest"
   "testing"
)

func TestParseRule(t *testing.T) {
	tests := []struct {
		name             string
		ruleLine         string
		expectedMsg      string
		expectedSID      string
		expectedContents []ContentMatch
		expectError      bool
	}{
		{
			name: "Valid Rule with Two Contents",
			ruleLine: `alert http $EXTERNAL_NET any -> $HTTP_SERVERS any (
                msg:"ET WEB_SERVER SQL Injection BULK INSERT in URI to Insert File Content into Database Table";
                flow:established,to_server;
                http.uri;
                content:"BULK"; nocase;
                content:"INSERT"; nocase; distance:0;
                sid:2011035;
            )`,
			expectedMsg: "ET WEB_SERVER SQL Injection BULK INSERT in URI to Insert File Content into Database Table",
			expectedSID: "2011035",
			expectedContents: []ContentMatch{
				{
					Buffer:    "http.uri",
					Pattern:   "BULK",
					Modifiers: ContentModifiers{NoCase: true},
				},
				{
					Buffer:    "http.uri",
					Pattern:   "INSERT",
					Modifiers: ContentModifiers{NoCase: true, Distance: 0},
				},
			},
			expectError: false,
		},
		{
			name: "Valid Rule with Multiple Contents",
			ruleLine: `alert http $HOME_NET any -> $EXTERNAL_NET any (
                msg:"ET WEB_SERVER .PHP being served from WP 1-flash-gallery Upload DIR (likely malicious)";
                flow:established,to_server;
                http.uri;
                content:"/wp-content/uploads/fgallery/"; nocase;
                content:".php"; nocase; distance:0;
                sid:2015518;
            )`,
			expectedMsg: "ET WEB_SERVER .PHP being served from WP 1-flash-gallery Upload DIR (likely malicious)",
			expectedSID: "2015518",
			expectedContents: []ContentMatch{
				{
					Buffer:    "http.uri",
					Pattern:   "/wp-content/uploads/fgallery/",
					Modifiers: ContentModifiers{NoCase: true},
				},
				{
					Buffer:    "http.uri",
					Pattern:   ".php",
					Modifiers: ContentModifiers{NoCase: true, Distance: 0},
				},
			},
			expectError: false,
		},
		{
			name: "Invalid Rule Missing Content",
			ruleLine: `alert http $EXTERNAL_NET any -> $HTTP_SERVERS any (
                msg:"Invalid rule";
                flow:established,to_server;
                http.uri;
            )`,
			expectedMsg:      "",
			expectedSID:      "",
			expectedContents: nil,
			expectError:      true,
		},
		{
			name: "Invalid Rule Missing Msg",
			ruleLine: `alert http $EXTERNAL_NET any -> $HTTP_SERVERS any (
                flow:established,to_server;
                http.uri;
                content:"test"; nocase;
                sid:1000002;
            )`,
			expectedMsg:      "",
			expectedSID:      "",
			expectedContents: nil,
			expectError:      true,
		},
		{
			name: "Invalid Rule Missing SID",
			ruleLine: `alert http $EXTERNAL_NET any -> $HTTP_SERVERS any (
                msg:"Missing SID";
                flow:established,to_server;
                http.uri;
                content:"test"; nocase;
            )`,
			expectedMsg:      "",
			expectedSID:      "",
			expectedContents: nil,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			rule, err := parseRule(tt.ruleLine)
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify rule fields
			if rule.Msg != tt.expectedMsg {
				t.Errorf("Expected msg '%s', got '%s'", tt.expectedMsg, rule.Msg)
			}
			if rule.SID != tt.expectedSID {
				t.Errorf("Expected SID '%s', got '%s'", tt.expectedSID, rule.SID)
			}

			// Verify content matches
			if len(rule.Contents) != len(tt.expectedContents) {
				t.Fatalf("Expected %d content matches, got %d", len(tt.expectedContents), len(rule.Contents))
			}

			for i, expectedContent := range tt.expectedContents {
				actualContent := rule.Contents[i]
				if actualContent.Pattern != expectedContent.Pattern {
					t.Errorf("Content %d: Expected pattern '%s', got '%s'", i+1, expectedContent.Pattern, actualContent.Pattern)
				}
				if actualContent.Modifiers.NoCase != expectedContent.Modifiers.NoCase {
					t.Errorf("Content %d: Expected NoCase '%v', got '%v'", i+1, expectedContent.Modifiers.NoCase, actualContent.Modifiers.NoCase)
				}
				if actualContent.Modifiers.Distance != expectedContent.Modifiers.Distance {
					t.Errorf("Content %d: Expected Distance '%d', got '%d'", i+1, expectedContent.Modifiers.Distance, actualContent.Modifiers.Distance)
				}
				if actualContent.Modifiers.Within != expectedContent.Modifiers.Within {
					t.Errorf("Content %d: Expected Within '%d', got '%d'", i+1, expectedContent.Modifiers.Within, actualContent.Modifiers.Within)
				}
			}
		})
	}
}

func TestExtractContent(t *testing.T) {
	tests := []struct {
		name        string
		option      string
		expected    string
		expectError bool
	}{
		{
			name:        "Valid Content",
			option:      `content:"/test-path";`,
			expected:    "/test-path",
			expectError: false,
		},
		{
			name:        "Invalid Content Missing Quotes",
			option:      `content:/invalid-content/;`,
			expected:    "",
			expectError: true,
		},
		{
			name:        "Valid Content with Special Characters",
			option:      `content:"$pecial-Char*()";`,
			expected:    "$pecial-Char*()",
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractContent(tt.option)
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestExtractModifierValue(t *testing.T) {
	tests := []struct {
		name        string
		option      string
		key         string
		expected    int
		expectError bool
	}{
		{
			name:        "Valid Distance Modifier",
			option:      `distance:5`,
			key:         "distance",
			expected:    5,
			expectError: false,
		},
		{
			name:        "Invalid Distance Modifier",
			option:      `distance:five`,
			key:         "distance",
			expected:    0,
			expectError: true,
		},
		{
			name:        "Valid Within Modifier",
			option:      `within:10`,
			key:         "within",
			expected:    10,
			expectError: false,
		},
		{
			name:        "Invalid Within Modifier",
			option:      `within:ten`,
			key:         "within",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractModifierValue(tt.option, tt.key)
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected '%d', got '%d'", tt.expected, result)
			}
		})
	}
}

func TestExtractQuotedString(t *testing.T) {
	tests := []struct {
		name        string
		option      string
		expected    string
		expectError bool
	}{
		{
			name:        "Valid Quoted String",
			option:      `msg:"Some message";`,
			expected:    "Some message",
			expectError: false,
		},
		{
			name:        "Invalid Quoted String Missing Quotes",
			option:      `msg:Some message;`,
			expected:    "",
			expectError: true,
		},
		{
			name:        "Empty Quoted String",
			option:      `msg:"";`,
			expected:    "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractQuotedString(tt.option)
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
// TestDecodeHexPattern verifies decoding of hex patterns enclosed in pipes.
func TestDecodeHexPattern(t *testing.T) {
   tests := []struct{
       name    string
       raw     string
       want    string
       wantErr bool
   }{
       {"No hex", "abc", "abc", false},
       {"Simple hex", "|41|", "A", false},
       {"Mixed", "foo|3b|bar", "foo;bar", false},
       {"Multi-byte", "|41 42|", "AB", false},
       {"Multi segments", "x|20|y|21|z", "x y!z", false},
       {"Unterminated", "foo|41", "", true},
       {"Invalid byte", "|zz|", "", true},
   }
   for _, tt := range tests {
       got, err := decodeHexPattern(tt.raw)
       if tt.wantErr {
           if err == nil {
               t.Errorf("%s: expected error, got none", tt.name)
           }
           continue
       }
       if err != nil {
           t.Errorf("%s: unexpected error: %v", tt.name, err)
           continue
       }
       if got != tt.want {
           t.Errorf("%s: expected %q, got %q", tt.name, tt.want, got)
       }
   }
}

// TestExtractPcre verifies parsing and compilation of PCRE patterns.
func TestExtractPcre(t *testing.T) {
   tests := []struct{
       name    string
       option  string
       wantPat string
       wantErr bool
   }{
       {"Valid simple", `pcre:"/foo/"`, "foo", false},
       {"Valid flags", `pcre:"/Bar/i"`, "(?i)Bar", false},
       {"Invalid prefix", `pcr:"/foo/"`, "", true},
       {"Missing slashes", `pcre:"foo"`, "", true},
       {"Bad regex", `pcre:"/(/"`, "", true},
   }
   for _, tt := range tests {
       re, _, _, err := extractPcre(tt.option)
       if tt.wantErr {
           if err == nil {
               t.Errorf("%s: expected error, got none", tt.name)
           }
           continue
       }
       if err != nil {
           t.Errorf("%s: unexpected error: %v", tt.name, err)
           continue
       }
       if re.String() != tt.wantPat {
           t.Errorf("%s: expected pattern %q, got %q", tt.name, tt.wantPat, re.String())
       }
   }
}
// TestParseRuleHex ensures that parseRule decodes hex sequences in content patterns.
func TestParseRuleHex(t *testing.T) {
   ruleLine := `alert http $EXTERNAL_NET any -> $HOME_NET any (
       msg:"HexTest";
       flow:established,to_server;
       http.uri;
       content:"foo|3b|bar"; nocase;
       sid:1234;
   )`
   rule, err := parseRule(ruleLine)
   if err != nil {
       t.Fatalf("Unexpected error: %v", err)
   }
   if rule.Msg != "HexTest" {
       t.Errorf("Expected Msg 'HexTest', got '%s'", rule.Msg)
   }
   if rule.SID != "1234" {
       t.Errorf("Expected SID '1234', got '%s'", rule.SID)
   }
   if len(rule.Contents) != 1 {
       t.Fatalf("Expected 1 content match, got %d", len(rule.Contents))
   }
   cm := rule.Contents[0]
   if cm.Pattern != "foo;bar" {
       t.Errorf("Expected decoded pattern 'foo;bar', got '%s'", cm.Pattern)
   }
   if !cm.Modifiers.NoCase {
       t.Errorf("Expected NoCase true, got false")
   }
}

// TestParseRulePcre ensures that parseRule extracts and compiles PCRE patterns.
func TestParseRulePcre(t *testing.T) {
   ruleLine := `alert http $EXTERNAL_NET any -> $HTTP_SERVERS any (
       msg:"PcreTest";
       flow:established,to_server;
       http.uri;
       pcre:"/abc[0-9]+/i";
       sid:5678;
   )`
   rule, err := parseRule(ruleLine)
   if err != nil {
       t.Fatalf("Unexpected error: %v", err)
   }
   if rule.Msg != "PcreTest" {
       t.Errorf("Expected Msg 'PcreTest', got '%s'", rule.Msg)
   }
   if rule.SID != "5678" {
       t.Errorf("Expected SID '5678', got '%s'", rule.SID)
   }
   if len(rule.Pcre) != 1 {
       t.Fatalf("Expected 1 PCRE match, got %d", len(rule.Pcre))
   }
   rm := rule.Pcre[0]
   if rm.Buffer != "http.uri" {
       t.Errorf("Expected PCRE Buffer 'http.uri', got '%s'", rm.Buffer)
   }
   if !rm.Regexp.MatchString("abc123") {
       t.Errorf("Expected regex to match 'abc123'")
   }
   // Check that case-insensitive flag applied
   if !rm.Regexp.MatchString("ABC123") {
       t.Errorf("Expected regex to match 'ABC123' with ignore-case flag")
   }
}

func TestSplitOptions(t *testing.T) {
	tests := []struct {
		name     string
		options  string
		expected []string
	}{
		{
			name:    "Standard Options",
			options: `msg:"Test rule"; content:"/test"; nocase; distance:3;`,
			expected: []string{
				`msg:"Test rule"`,
				`content:"/test"`,
				`nocase`,
				`distance:3`,
			},
		},
		{
			name:    "Options with Semicolons Inside Quotes",
			options: `msg:"Rule; with semicolon"; content:"/test;path/"; nocase;`,
			expected: []string{
				`msg:"Rule; with semicolon"`,
				`content:"/test;path/"`,
				`nocase`,
			},
		},
		{
			name:     "Empty Options",
			options:  ``,
			expected: []string{},
		},
		{
			name:     "Only Comments and Whitespaces",
			options:  `   ;  ; `,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			result := splitOptions(tt.options)
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d options, got %d", len(tt.expected), len(result))
			}
			for i, opt := range tt.expected {
				if result[i] != opt {
					t.Errorf("Expected option '%s', got '%s'", opt, result[i])
				}
			}
		})
	}
}

func TestMatch(t *testing.T) {
   tests := []struct {
       name         string
       rules        []Rule
       uri          string
       expectedSIDs []string
   }{
		{
			name: "Single Rule Match",
			rules: []Rule{
				{
					Msg: "Test Rule",
					SID: "1001",
           Contents: []ContentMatch{
               {
                   Buffer:    "http.uri",
                   Pattern:   "/test",
                   Modifiers: ContentModifiers{NoCase: true},
               },
           },
				},
			},
			uri:          "/Test/Path",
			expectedSIDs: []string{"1001"},
		},
		{
			name: "Single Rule No Match",
			rules: []Rule{
				{
					Msg: "Test Rule",
					SID: "1001",
           Contents: []ContentMatch{
               {
                   Buffer:    "http.uri",
                   Pattern:   "/test",
                   Modifiers: ContentModifiers{NoCase: true},
               },
           },
				},
			},
			uri:          "/no/match/here",
			expectedSIDs: []string{},
		},
		{
			name: "Multiple Rules with One Match",
			rules: []Rule{
				{
					Msg: "Rule One",
					SID: "1001",
           Contents: []ContentMatch{
               {
                   Buffer:    "http.uri",
                   Pattern:   "/test",
                   Modifiers: ContentModifiers{NoCase: true},
               },
               {
                   Buffer:    "http.uri",
                   Pattern:   "insert",
                   Modifiers: ContentModifiers{NoCase: true},
               },
           },
				},
				{
					Msg: "Rule Two",
					SID: "1002",
           Contents: []ContentMatch{
               {
                   Buffer:    "http.uri",
                   Pattern:   "/admin",
                   Modifiers: ContentModifiers{NoCase: false},
               },
           },
				},
			},
			uri:          "/Test/Insert/admin",
			expectedSIDs: []string{"1001", "1002"},
		},
		{
			name: "Multiple Contents No Match",
			rules: []Rule{
				{
					Msg: "Rule One",
					SID: "1001",
           Contents: []ContentMatch{
               {
                   Buffer:    "http.uri",
                   Pattern:   "/test",
                   Modifiers: ContentModifiers{NoCase: true},
               },
               {
                   Buffer:    "http.uri",
                   Pattern:   "insert",
                   Modifiers: ContentModifiers{NoCase: true},
               },
           },
				},
			},
			uri:          "/test/path",
			expectedSIDs: []string{},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
           rs := NewRuleSet()
           rs.Rules = append(rs.Rules, tt.rules...)
           // build a dummy HTTP request for matching
           req := httptest.NewRequest("GET", tt.uri, nil)
           // ensure RequestURI is set
           req.RequestURI = tt.uri
           matches := rs.Match(req, "")
			if len(matches) != len(tt.expectedSIDs) {
				t.Fatalf("Expected %d matches, got %d", len(tt.expectedSIDs), len(matches))
			}

			for i, expectedSID := range tt.expectedSIDs {
				if matches[i].SID != expectedSID {
					t.Errorf("Expected SID '%s', got '%s'", expectedSID, matches[i].SID)
				}
			}
		})
	}
}
