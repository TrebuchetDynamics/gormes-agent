package parser

type jsonStringState struct {
	inString bool
	escaped  bool
}

func (s jsonStringState) InString() bool {
	return s.inString
}

func (s *jsonStringState) Consume(ch byte) {
	if !s.inString {
		if ch == '"' {
			s.inString = true
		}
		return
	}
	if s.escaped {
		s.escaped = false
		return
	}
	if ch == '\\' {
		s.escaped = true
		return
	}
	if ch == '"' {
		s.inString = false
	}
}
