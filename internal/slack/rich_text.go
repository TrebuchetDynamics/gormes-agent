package slack

import (
	"fmt"
	"strings"
)

const slackRichTextUnavailableCode = "slack_rich_text_unavailable"

type SlackBlock map[string]any

type SlackAttachmentPreview map[string]any

type SlackRichTextEvidence struct {
	Code   string
	Source string
	Reason string
}

func augmentInboundText(text string, blocks []SlackBlock, attachments []SlackAttachmentPreview) (string, []SlackRichTextEvidence) {
	base := strings.TrimSpace(text)
	evidence := []SlackRichTextEvidence{}

	out := base
	seen := lineSet(splitNonEmptyLines(base))
	for _, line := range renderSlackRichTextBlocks(blocks, &evidence) {
		normalized := strings.TrimSpace(line)
		if normalized == "" || seen[normalized] {
			continue
		}
		if out == "" {
			out = line
		} else {
			out += "\n" + line
		}
		seen[normalized] = true
	}

	for _, section := range renderSlackAttachmentPreviews(attachments, &evidence) {
		if section == "" || strings.Contains(out, section) {
			continue
		}
		if out == "" {
			out = section
			continue
		}
		out += "\n\n" + section
	}

	if out == "" {
		out = base
	}
	return out, evidence
}

func renderSlackRichTextBlocks(blocks []SlackBlock, evidence *[]SlackRichTextEvidence) []string {
	lines := []string{}
	for i, block := range blocks {
		blockType, ok := optionalString(block, "type", "blocks", evidence)
		if !ok {
			continue
		}
		if blockType != "rich_text" {
			continue
		}
		elements, ok := blockElements(block, "elements", "blocks", evidence)
		if !ok {
			addUnavailable(evidence, "blocks", fmt.Sprintf("rich_text block %d elements are not parseable", i))
			continue
		}
		walkRichTextElements(elements, 0, "", evidence, &lines)
	}
	return lines
}

func walkRichTextElements(elements []SlackBlock, quoteDepth int, bullet string, evidence *[]SlackRichTextEvidence, lines *[]string) {
	for _, elem := range elements {
		elemType, ok := optionalString(elem, "type", "blocks", evidence)
		if !ok {
			continue
		}

		switch elemType {
		case "rich_text_section":
			children, ok := blockElements(elem, "elements", "blocks", evidence)
			if !ok {
				addUnavailable(evidence, "blocks", "rich_text_section elements are not parseable")
				continue
			}
			appendRenderedLine(lines, renderInlineElements(children, evidence), quoteDepth, bullet)
		case "rich_text_quote":
			children, ok := blockElements(elem, "elements", "blocks", evidence)
			if !ok {
				addUnavailable(evidence, "blocks", "rich_text_quote elements are not parseable")
				continue
			}
			walkRichTextElements(children, quoteDepth+1, "", evidence, lines)
		case "rich_text_list":
			children, ok := blockElements(elem, "elements", "blocks", evidence)
			if !ok {
				addUnavailable(evidence, "blocks", "rich_text_list elements are not parseable")
				continue
			}
			style, ok := optionalString(elem, "style", "blocks", evidence)
			if !ok {
				continue
			}
			for i, child := range children {
				itemBullet := "- "
				if style == "ordered" {
					itemBullet = fmt.Sprintf("%d. ", i+1)
				}
				walkRichTextElements([]SlackBlock{child}, quoteDepth, itemBullet, evidence, lines)
			}
		case "rich_text_preformatted":
			children, ok := blockElements(elem, "elements", "blocks", evidence)
			if !ok {
				addUnavailable(evidence, "blocks", "rich_text_preformatted elements are not parseable")
				continue
			}
			codeLines := []string{}
			for _, child := range children {
				childType, ok := optionalString(child, "type", "blocks", evidence)
				if !ok {
					continue
				}
				if childType == "rich_text_section" {
					sectionChildren, ok := blockElements(child, "elements", "blocks", evidence)
					if !ok {
						addUnavailable(evidence, "blocks", "preformatted section elements are not parseable")
						continue
					}
					if rendered := renderInlineElements(sectionChildren, evidence); strings.TrimSpace(rendered) != "" {
						codeLines = append(codeLines, rendered)
					}
					continue
				}
				if rendered := renderInlineElements([]SlackBlock{child}, evidence); strings.TrimSpace(rendered) != "" {
					codeLines = append(codeLines, rendered)
				}
			}
			if len(codeLines) == 0 {
				continue
			}
			language, ok := optionalString(elem, "language", "blocks", evidence)
			if !ok {
				continue
			}
			appendRenderedLine(lines, "```"+language+"\n"+strings.Join(codeLines, "\n")+"\n```", quoteDepth, bullet)
		default:
			appendRenderedLine(lines, renderInlineElements([]SlackBlock{elem}, evidence), quoteDepth, bullet)
		}
	}
}

func renderInlineElements(elements []SlackBlock, evidence *[]SlackRichTextEvidence) string {
	pieces := []string{}
	for _, elem := range elements {
		elemType, ok := optionalString(elem, "type", "blocks", evidence)
		if !ok {
			continue
		}
		switch elemType {
		case "text":
			text, ok := optionalString(elem, "text", "blocks", evidence)
			if ok {
				pieces = append(pieces, text)
			}
		case "link":
			url, okURL := optionalString(elem, "url", "blocks", evidence)
			label, okLabel := optionalString(elem, "text", "blocks", evidence)
			if !okURL || !okLabel {
				continue
			}
			switch {
			case label != "" && url != "" && label != url:
				pieces = append(pieces, label+" ("+url+")")
			case label != "":
				pieces = append(pieces, label)
			default:
				pieces = append(pieces, url)
			}
		case "channel":
			if id, ok := optionalString(elem, "channel_id", "blocks", evidence); ok && id != "" {
				pieces = append(pieces, "<#"+id+">")
			}
		case "user":
			if id, ok := optionalString(elem, "user_id", "blocks", evidence); ok && id != "" {
				pieces = append(pieces, "<@"+id+">")
			}
		case "usergroup":
			if id, ok := optionalString(elem, "usergroup_id", "blocks", evidence); ok && id != "" {
				pieces = append(pieces, "<!subteam^"+id+">")
			}
		case "emoji":
			if name, ok := optionalString(elem, "name", "blocks", evidence); ok && name != "" {
				pieces = append(pieces, ":"+name+":")
			}
		case "broadcast":
			scope, ok := optionalString(elem, "range", "blocks", evidence)
			if !ok {
				continue
			}
			if scope == "" {
				scope = "here"
			}
			pieces = append(pieces, "<!"+scope+">")
		case "date":
			if fallback, ok := optionalString(elem, "fallback", "blocks", evidence); ok {
				pieces = append(pieces, fallback)
			}
		default:
			if text, ok := optionalString(elem, "text", "blocks", evidence); ok {
				pieces = append(pieces, text)
			}
		}
	}
	return strings.Join(pieces, "")
}

func renderSlackAttachmentPreviews(attachments []SlackAttachmentPreview, evidence *[]SlackRichTextEvidence) []string {
	sections := []string{}
	for i, attachment := range attachments {
		isMsgUnfurl, ok := optionalBool(attachment, "is_msg_unfurl", "attachments", evidence)
		if !ok {
			addUnavailable(evidence, "attachments", fmt.Sprintf("attachment %d is_msg_unfurl is not parseable", i))
			continue
		}
		if isMsgUnfurl {
			continue
		}

		title, titleOK := optionalString(attachment, "title", "attachments", evidence)
		titleLink, titleLinkOK := optionalString(attachment, "title_link", "attachments", evidence)
		fromURL, fromURLOK := optionalString(attachment, "from_url", "attachments", evidence)
		body, bodyOK := optionalString(attachment, "text", "attachments", evidence)
		fallback, fallbackOK := optionalString(attachment, "fallback", "attachments", evidence)
		footer, footerOK := optionalString(attachment, "footer", "attachments", evidence)
		if !titleOK || !titleLinkOK || !fromURLOK || !bodyOK || !fallbackOK || !footerOK {
			addUnavailable(evidence, "attachments", fmt.Sprintf("attachment %d preview fields are not parseable", i))
			continue
		}

		url := titleLink
		if url == "" {
			url = fromURL
		}
		if body == "" {
			body = fallback
		}

		lines := []string{}
		if title != "" {
			lines = append(lines, "Link preview: "+title)
		} else if url != "" || body != "" || footer != "" {
			lines = append(lines, "Link preview:")
		}
		if url != "" {
			lines = append(lines, url)
		}
		if body != "" {
			lines = append(lines, body)
		}
		if footer != "" {
			lines = append(lines, footer)
		}
		if len(lines) > 0 {
			sections = append(sections, strings.Join(lines, "\n"))
		}
	}
	return sections
}

func appendRenderedLine(lines *[]string, text string, quoteDepth int, bullet string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	prefix := ""
	if quoteDepth > 0 {
		prefix = strings.Repeat("> ", quoteDepth)
	}
	renderedLines := strings.Split(text, "\n")
	for i := range renderedLines {
		if i == 0 {
			renderedLines[i] = prefix + bullet + renderedLines[i]
			continue
		}
		renderedLines[i] = prefix + renderedLines[i]
	}
	*lines = append(*lines, strings.Join(renderedLines, "\n"))
}

func splitNonEmptyLines(text string) []string {
	lines := []string{}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func lineSet(lines []string) map[string]bool {
	seen := make(map[string]bool, len(lines))
	for _, line := range lines {
		if normalized := strings.TrimSpace(line); normalized != "" {
			seen[normalized] = true
		}
	}
	return seen
}

func blockElements(block SlackBlock, key, source string, evidence *[]SlackRichTextEvidence) ([]SlackBlock, bool) {
	value, ok := block[key]
	if !ok || value == nil {
		return nil, true
	}
	return blockList(value, source, evidence)
}

func blockList(value any, source string, evidence *[]SlackRichTextEvidence) ([]SlackBlock, bool) {
	switch typed := value.(type) {
	case []SlackBlock:
		return typed, true
	case []map[string]any:
		out := make([]SlackBlock, 0, len(typed))
		for _, item := range typed {
			out = append(out, SlackBlock(item))
		}
		return out, true
	case []any:
		out := make([]SlackBlock, 0, len(typed))
		for _, item := range typed {
			block, ok := asSlackBlock(item)
			if !ok {
				addUnavailable(evidence, source, fmt.Sprintf("element has unsupported type %T", item))
				return nil, false
			}
			out = append(out, block)
		}
		return out, true
	default:
		addUnavailable(evidence, source, fmt.Sprintf("elements have unsupported type %T", value))
		return nil, false
	}
}

func asSlackBlock(value any) (SlackBlock, bool) {
	switch typed := value.(type) {
	case SlackBlock:
		return typed, true
	case map[string]any:
		return SlackBlock(typed), true
	default:
		return nil, false
	}
}

func optionalString(values map[string]any, key, source string, evidence *[]SlackRichTextEvidence) (string, bool) {
	value, ok := values[key]
	if !ok || value == nil {
		return "", true
	}
	text, ok := value.(string)
	if !ok {
		addUnavailable(evidence, source, fmt.Sprintf("%s has unsupported type %T", key, value))
		return "", false
	}
	return text, true
}

func optionalBool(values map[string]any, key, source string, evidence *[]SlackRichTextEvidence) (bool, bool) {
	value, ok := values[key]
	if !ok || value == nil {
		return false, true
	}
	flag, ok := value.(bool)
	if !ok {
		addUnavailable(evidence, source, fmt.Sprintf("%s has unsupported type %T", key, value))
		return false, false
	}
	return flag, true
}

func addUnavailable(evidence *[]SlackRichTextEvidence, source, reason string) {
	*evidence = append(*evidence, SlackRichTextEvidence{
		Code:   slackRichTextUnavailableCode,
		Source: source,
		Reason: reason,
	})
}
