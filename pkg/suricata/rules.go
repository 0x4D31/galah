package suricata

import (
   "bufio"
   "fmt"
   "net/http"
   "os"
   "path/filepath"
   "regexp"
   "strconv"
   "strings"
   "sync"
)

// ContentModifiers represents modifiers applied to a content match.
type ContentModifiers struct {
	NoCase   bool // Indicates if the content match should be case-insensitive.
	Distance int  // Specifies the number of bytes between this content match and the previous one.
	Within   int  // Specifies the maximum number of bytes between this content match and the previous one.
}

// ContentMatch represents a content pattern and its associated modifiers.
type ContentMatch struct {
   Buffer    string           // Buffer to match (e.g., http.method, http.uri, http.request_body).
   Pattern   string           // The content pattern to match in the buffer.
   Modifiers ContentModifiers // Modifiers affecting the content match.
   Negative  bool             // Whether this is a negated content match (content:!).
}
// RegexMatch represents a PCRE pattern and its associated buffer.
type RegexMatch struct {
   Buffer     string         // Buffer to match (e.g., http.method, http.uri, http.request_body).
   Regexp     *regexp.Regexp // Compiled PCRE regex (flags applied except 'R').
   RawPattern string         // The raw regex pattern (without delimiters or flags).
   Flags      string         // Original PCRE flags string (may include 'i', 'R', etc.).
}

// Rule represents a parsed Suricata rule with relevant fields.
type Rule struct {
	Msg      string         // Message describing the rule.
	SID      string         // Unique identifier for the rule.
	Contents []ContentMatch // List of content patterns to match in the URI or body.
	Pcre     []RegexMatch   // List of PCRE regex patterns to match in the URI or body.
}

// RuleSet holds all parsed rules and provides thread-safe access.
type RuleSet struct {
	Rules []Rule       // Slice of parsed rules.
	mutex sync.RWMutex // Mutex to ensure thread-safe access.
}

// NewRuleSet initializes and returns an empty RuleSet.
func NewRuleSet() *RuleSet {
	return &RuleSet{
		Rules: make([]Rule, 0),
	}
}

// LoadRules loads and parses all Suricata rules from the specified directory.
// It only retains rules containing the 'http.uri' keyword with associated content patterns.
func (rs *RuleSet) LoadRules(directory string) error {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	// Read all files in the directory.
	files, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("failed to read directory '%s': %w", directory, err)
	}

	for _, file := range files {
		// Process only files with a '.rules' extension.
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".rules") {
			path := filepath.Join(directory, file.Name())
			if err := rs.parseRuleFile(path); err != nil {
				return fmt.Errorf("error parsing file '%s': %w", path, err)
			}
		}
	}
	return nil
}

// parseRuleFile parses a single Suricata rule file.
func (rs *RuleSet) parseRuleFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file '%s': %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
   for scanner.Scan() {
       lineNumber++
       line := strings.TrimSpace(scanner.Text())
       // Skip empty lines and comments.
       if line == "" || strings.HasPrefix(line, "#") {
           continue
       }
       // Only process lines containing supported HTTP buffer keywords
       if !(strings.Contains(line, "http.uri") || strings.Contains(line, "http.method") || strings.Contains(line, "http.request_body")) {
           continue
       }
       rule, err := parseRule(line)
       if err != nil {
           // Skip rules that cannot be parsed (e.g., unsupported content syntax)
           continue
       }
      // Retain rules with at least one content or PCRE match
      if len(rule.Contents) > 0 || len(rule.Pcre) > 0 {
          rs.Rules = append(rs.Rules, rule)
      }
   }

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file '%s': %w", path, err)
	}
	return nil
}

// parseRule parses a single Suricata rule line and extracts relevant fields.
func parseRule(line string) (Rule, error) {
	var rule Rule
	// Split the rule into header and options.
	parts := strings.SplitN(line, "(", 2)
	if len(parts) != 2 {
		return rule, fmt.Errorf("invalid rule format: missing options")
	}
	optionsPart := strings.TrimSuffix(strings.TrimSpace(parts[1]), ")")
	options := splitOptions(optionsPart)

	var currentBuffer string
	var contentMatches []ContentMatch
	var lastContentMatch *ContentMatch
	// collect regex (PCRE) matches
	var regexMatches []RegexMatch

	for _, opt := range options {
		opt = strings.TrimSpace(opt)

		if isBufferModifier(opt) {
			currentBuffer = opt
			// reset last content match when buffer changes
			lastContentMatch = nil
			continue
		}

		if isRuleOption(opt) {
			if strings.HasPrefix(opt, "msg:") {
				msg, err := extractQuotedString(opt)
				if err != nil {
					return rule, err
				}
				rule.Msg = msg
			}
			if strings.HasPrefix(opt, "sid:") {
				sid := strings.TrimPrefix(opt, "sid:")
				sid = strings.TrimSpace(strings.TrimSuffix(sid, ";"))
				rule.SID = sid
			}
			// Reset lastContentMatch because it's a rule option.
			lastContentMatch = nil
			continue
		}

		if isModifierOption(opt) {
			if lastContentMatch != nil {
				err := applyModifier(lastContentMatch, opt)
				if err != nil {
					return rule, err
				}
			}
			continue
		}
       // PCRE regex match
       if strings.HasPrefix(opt, "pcre:") {
           // support PCRE on HTTP buffers
           switch currentBuffer {
           case "http.method", "http.uri", "http.request_body",
               "http.host", "http.host.raw", "http.cookie",
               "http.header", "http.header.raw", "http.header_names", "http.user_agent",
               "http.accept", "http.accept_enc", "http.accept_lang",
               "http.content_type", "http.protocol", "http.start":
               reCompiled, rawPat, flags, err := extractPcre(opt)
               if err != nil {
                   // skip invalid PCRE patterns
                   break
               }
               regexMatches = append(regexMatches, RegexMatch{
                   Buffer:     currentBuffer,
                   Regexp:     reCompiled,
                   RawPattern: rawPat,
                   Flags:      flags,
               })
           }
           // reset content context
           lastContentMatch = nil
           continue
       }

       if strings.HasPrefix(opt, "content:") {
           // capture content for supported HTTP buffers, detect negation
           negative := false
           contentOpt := opt
           if strings.HasPrefix(opt, "content:!") {
               negative = true
               // rewrite to standard content: prefix
               contentOpt = "content:" + opt[len("content:!"):]
           }
           switch currentBuffer {
           case "http.method", "http.uri", "http.request_body",
               "http.host", "http.host.raw", "http.cookie",
               "http.header", "http.header.raw", "http.header_names", "http.user_agent",
               "http.accept", "http.accept_enc", "http.accept_lang",
               "http.content_type", "http.protocol", "http.start":
               pattern, err := extractContent(contentOpt)
               if err != nil {
                   lastContentMatch = nil
                   continue
               }
               cm := ContentMatch{
                   Buffer:    currentBuffer,
                   Pattern:   pattern,
                   Modifiers: ContentModifiers{},
                   Negative:  negative,
               }
               contentMatches = append(contentMatches, cm)
               lastContentMatch = &contentMatches[len(contentMatches)-1]
           default:
               lastContentMatch = nil
           }
           continue
       }

		// Other options like flow, reference, classtype, etc., are ignored
	}

   // Require at least one supported content match
   if len(contentMatches) == 0 && len(regexMatches) == 0 {
       return rule, fmt.Errorf("no supported buffer content or PCRE options found")
   }

   if rule.Msg == "" || rule.SID == "" {
       return rule, fmt.Errorf("rule missing 'msg' or 'sid'")
   }

   rule.Contents = contentMatches
  	// attach any PCRE regex matches
   rule.Pcre = regexMatches
   return rule, nil
}

// isBufferModifier checks if the option is a buffer modifier.
func isBufferModifier(opt string) bool {
   bufferModifiers := []string{
       // request buffers
       "http.method",
       "http.uri",
       "http.request_body",
       "http.host",
       "http.host.raw",
       "http.cookie",
       "http.header",
       "http.header.raw",
       // header names buffer (just names, CRLF separated)
       "http.header_names",
       "http.user_agent",
       "http.accept",
       "http.accept_enc",
       "http.accept_lang",
       "http.content_type",
       "http.protocol",
       "http.start",
       // legacy or response buffers (not yet fully supported)
       "http_request_line",
       "http_response_line",
   }
	for _, bm := range bufferModifiers {
		if opt == bm {
			return true
		}
	}
	return false
}

// isRuleOption checks if the option is a known rule field that sets rule attributes.
func isRuleOption(opt string) bool {
	knownRuleOptions := []string{"msg:", "sid:"}
	for _, known := range knownRuleOptions {
		if strings.HasPrefix(opt, known) {
			return true
		}
	}
	return false
}

// isModifierOption checks if the option is a known modifier.
func isModifierOption(opt string) bool {
	knownModifiers := []string{"nocase", "distance:", "within:"}
	for _, mod := range knownModifiers {
		if strings.HasPrefix(opt, mod) {
			return true
		}
	}
	return false
}

// applyModifier applies the modifier to the given ContentMatch.
func applyModifier(cm *ContentMatch, opt string) error {
	switch {
	case opt == "nocase":
		cm.Modifiers.NoCase = true
	case strings.HasPrefix(opt, "distance:"):
		distance, err := extractModifierValue(opt, "distance")
		if err != nil {
			return err
		}
		cm.Modifiers.Distance = distance
	case strings.HasPrefix(opt, "within:"):
		within, err := extractModifierValue(opt, "within")
		if err != nil {
			return err
		}
		cm.Modifiers.Within = within
	default:
		// Unknown modifier; ignore or handle as needed
	}
	return nil
}

// splitOptions splits the options string into individual options, respecting quoted strings.
func splitOptions(options string) []string {
	var opts []string
	current := ""
	inQuotes := false
	for i := 0; i < len(options); i++ {
		char := options[i]
		switch char {
		case '"':
			inQuotes = !inQuotes
			current += string(char)
		case ';':
			if inQuotes {
				current += string(char)
			} else {
				trimmed := strings.TrimSpace(current)
				if trimmed != "" { // Only append if trimmed is not empty
					opts = append(opts, trimmed)
				}
				current = ""
			}
		default:
			current += string(char)
		}
	}
	// After loop ends, check if there's any remaining content
	trimmed := strings.TrimSpace(current)
	if trimmed != "" {
		opts = append(opts, trimmed)
	}
	return opts
}

// extractContent extracts the content pattern from a content option and decodes any hex sequences.
func extractContent(option string) (string, error) {
   // Expected format: content:"value"
   re := regexp.MustCompile(`content:"([^"]*)"`)
   matches := re.FindStringSubmatch(option)
   if len(matches) != 2 {
       return "", fmt.Errorf("invalid content format: '%s'", option)
   }
   raw := matches[1]
   // Decode any hex sequences (e.g., |3b 0a|)
   decoded, err := decodeHexPattern(raw)
   if err != nil {
       return "", err
   }
   return decoded, nil
}

// extractModifierValue extracts the integer value of a modifier (e.g., distance:3).
func extractModifierValue(option, key string) (int, error) {
	re := regexp.MustCompile(fmt.Sprintf(`%s:(\d+)`, key))
	matches := re.FindStringSubmatch(option)
	if len(matches) != 2 {
		return 0, fmt.Errorf("invalid %s format: '%s'", key, option)
	}
	return strconv.Atoi(matches[1])
}

// extractQuotedString extracts a quoted string from an option.
func extractQuotedString(option string) (string, error) {
	// Example: msg:"Some message";
	re := regexp.MustCompile(`"([^"]*)"`)

	matches := re.FindStringSubmatch(option)
	if len(matches) != 2 {
		return "", fmt.Errorf("invalid quoted string format: '%s'", option)
	}
	return matches[1], nil
}
// decodeHexPattern decodes hex patterns enclosed in pipe characters, e.g., "|3b 0a|" -> "\x3b\x0a".
func decodeHexPattern(s string) (string, error) {
   var bldr strings.Builder
   for i := 0; i < len(s); {
       if s[i] == '|' {
           // find closing '|'
           end := strings.Index(s[i+1:], "|")
           if end < 0 {
               return "", fmt.Errorf("unterminated hex pattern in '%s'", s)
           }
           end = i + 1 + end
           hexSeq := s[i+1 : end]
           parts := strings.Fields(hexSeq)
           for _, part := range parts {
               v, err := strconv.ParseUint(part, 16, 8)
               if err != nil {
                   return "", fmt.Errorf("invalid hex byte '%s' in '%s'", part, s)
               }
               bldr.WriteByte(byte(v))
           }
           i = end + 1
       } else {
           bldr.WriteByte(s[i])
           i++
       }
   }
   return bldr.String(), nil
}

// extractPcre parses a PCRE option like `pcre:"/pattern/flags"`,
// extracts raw pattern and flags, and compiles a Go regexp ignoring the 'R' flag.
// Returns (compiledRegexp, rawPattern, flags, error).
func extractPcre(option string) (*regexp.Regexp, string, string, error) {
   // strip prefix and quotes
   const prefix = "pcre:\""
   if !strings.HasPrefix(option, prefix) || !strings.HasSuffix(option, "\"") {
       return nil, "", "", fmt.Errorf("invalid pcre format: '%s'", option)
   }
   // Extract the /pattern/flags string inside the quotes
   content := option[len(prefix) : len(option)-1]
   if len(content) < 2 || content[0] != '/' {
       return nil, "", "", fmt.Errorf("invalid pcre pattern: '%s'", content)
   }
   // find last slash delimiting end of pattern
   rel := strings.LastIndex(content[1:], "/")
   if rel < 0 {
       return nil, "", "", fmt.Errorf("invalid pcre pattern: '%s'", content)
   }
   idx := rel + 1
   // content[0]=='/' and content[idx]=='/'
   rawPattern := content[1:idx]
   flags := ""
   if idx+1 < len(content) {
       flags = content[idx+1:]
   }
   // Prepare pattern for compilation: drop 'R' flag
   compileFlags := strings.ReplaceAll(flags, "R", "")
   pattern := rawPattern
   if strings.Contains(compileFlags, "i") {
       pattern = "(?i)" + pattern
   }
   re, err := regexp.Compile(pattern)
   if err != nil {
       return nil, rawPattern, flags, fmt.Errorf("invalid pcre syntax '%s': %w", pattern, err)
   }
   return re, rawPattern, flags, nil
}

// Match evaluates all loaded rules against the given HTTP request and request body.
// It returns a slice of Rules for which all content and PCRE patterns match (logical AND).
func (rs *RuleSet) Match(req *http.Request, body string) []Rule {
   rs.mutex.RLock()
   defer rs.mutex.RUnlock()

   // Precompute buffers
   method := req.Method
   uri := req.RequestURI
   hostRaw := req.Header.Get("Host")
   hostNorm := strings.ToLower(req.Host)
   cookie := req.Header.Get("Cookie")
   // Build header string (normalized)
   var hbldr strings.Builder
   for name, vals := range req.Header {
       for _, v := range vals {
           hbldr.WriteString(name)
           hbldr.WriteString(": ")
           hbldr.WriteString(v)
           hbldr.WriteString("\r\n")
       }
   }
   header := hbldr.String()
   userAgent := req.UserAgent()
   accept := req.Header.Get("Accept")
   acceptEnc := req.Header.Get("Accept-Encoding")
   acceptLang := req.Header.Get("Accept-Language")
   contentType := req.Header.Get("Content-Type")
   protocol := req.Proto
   // http.start: request line + headers + blank line
   start := fmt.Sprintf("%s %s %s\r\n%s\r\n", method, uri, protocol, header)

   // Build http.header_names buffer: CRLF-separated header names, terminated by blank line
   var headerNamesBuilder strings.Builder
   headerNamesBuilder.WriteString("\r\n")
   for name := range req.Header {
       headerNamesBuilder.WriteString(name)
       headerNamesBuilder.WriteString("\r\n")
   }
   headerNamesBuilder.WriteString("\r\n")
   headerNames := headerNamesBuilder.String()
   var matched []Rule
   for _, rule := range rs.Rules {
       matchedAll := true
       // Literal/hex content matches
       for _, cm := range rule.Contents {
           var buf string
           switch cm.Buffer {
           case "http.method": buf = method
           case "http.uri": buf = uri
           case "http.request_body": buf = body
           case "http.host.raw": buf = hostRaw
           case "http.host": buf = hostNorm
           case "http.cookie": buf = cookie
           case "http.header.raw", "http.header": buf = header
           case "http.header_names": buf = headerNames
           case "http.user_agent": buf = userAgent
           case "http.accept": buf = accept
           case "http.accept_enc": buf = acceptEnc
           case "http.accept_lang": buf = acceptLang
           case "http.content_type": buf = contentType
           case "http.protocol": buf = protocol
           case "http.start": buf = start
           default:
               matchedAll = false
           }
           if !matchedAll {
               break
           }
           // Substring match, with optional negation
           if cm.Buffer == "http.header" || cm.Buffer == "http.header.raw" || cm.Buffer == "http.header_names" {
               // header buffers: always treat matches as case-insensitive
               bufLower := strings.ToLower(buf)
               patLower := strings.ToLower(cm.Pattern)
               found := strings.Contains(bufLower, patLower)
               if !cm.Negative {
                   if !found {
                       matchedAll = false
                   }
               } else {
                   if found {
                       matchedAll = false
                   }
               }
           } else if !cm.Negative {
               if cm.Modifiers.NoCase {
                   if !strings.Contains(strings.ToLower(buf), strings.ToLower(cm.Pattern)) {
                       matchedAll = false
                   }
               } else {
                   if !strings.Contains(buf, cm.Pattern) {
                       matchedAll = false
                   }
               }
           } else {
               // negated content: must NOT appear
               if cm.Modifiers.NoCase {
                   if strings.Contains(strings.ToLower(buf), strings.ToLower(cm.Pattern)) {
                       matchedAll = false
                   }
               } else {
                   if strings.Contains(buf, cm.Pattern) {
                       matchedAll = false
                   }
               }
           }
           if !matchedAll {
               break
           }
       }
       // Regex (PCRE) matches
       if matchedAll {
           for _, rm := range rule.Pcre {
               var buf string
               switch rm.Buffer {
               case "http.method": buf = method
               case "http.uri": buf = uri
               case "http.request_body": buf = body
               case "http.host.raw", "http.host": buf = req.Host
               case "http.cookie": buf = cookie
               case "http.header.raw", "http.header": buf = header
               case "http.header_names": buf = headerNames
               case "http.user_agent": buf = userAgent
               case "http.accept": buf = accept
               case "http.accept_enc": buf = acceptEnc
               case "http.accept_lang": buf = acceptLang
               case "http.content_type": buf = contentType
               case "http.protocol": buf = protocol
               case "http.start": buf = start
               default:
                   matchedAll = false
               }
               if !matchedAll {
                   break
               }
               // Handle 'R' (relative) flag by scanning substring; else full-match
               if strings.Contains(rm.Flags, "R") {
                   // Perform unanchored search: remove leading '^' if present
                   pat := rm.RawPattern
                   if strings.HasPrefix(pat, "^") {
                       pat = pat[1:]
                   }
                   if strings.Contains(rm.Flags, "i") {
                       pat = "(?i)" + pat
                   }
                   re2, err := regexp.Compile(pat)
                   if err != nil || !re2.MatchString(buf) {
                       matchedAll = false
                   }
               } else {
                   if !rm.Regexp.MatchString(buf) {
                       matchedAll = false
                   }
               }
               if !matchedAll {
                   break
               }
           }
       }
       if matchedAll {
           matched = append(matched, rule)
       }
   }
   return matched
}
