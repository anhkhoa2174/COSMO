package intent

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rs/zerolog"

	"github.com/openai/openai-go"
)

// IntentClassifier classifies email content into intent types using OpenAI
type IntentClassifier struct {
	client *openai.Client
	model  string
	logger *zerolog.Logger
}

// NewIntentClassifier creates a new intent classifier
func NewIntentClassifier(client *openai.Client, model string, logger *zerolog.Logger) *IntentClassifier {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &IntentClassifier{
		client: client,
		model:  model,
		logger: logger,
	}
}

// Classify classifies an email's intent synchronously
func (ic *IntentClassifier) Classify(ctx context.Context, emailContent string) (domain.IntentType, error) {
	prompt := ic.constructPrompt()

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(prompt),
		openai.UserMessage(fmt.Sprintf(`Here's the email you need to analyze:

<email_to_classify>
%s
</email_to_classify>`, emailContent)),
	}

	resp, err := ic.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModelGPT4oMini,
		Messages: messages,
	})

	if err != nil {
		ic.logger.Error().Err(err).Msg("Failed to classify email intent")
		return domain.IntentUnknown, fmt.Errorf("failed to classify intent: %w", err)
	}

	if len(resp.Choices) == 0 {
		return domain.IntentUnknown, fmt.Errorf("no response from OpenAI")
	}

	content := resp.Choices[0].Message.Content
	intentStr := ic.extractIntent(content)

	ic.logger.Info().Str("extracted_intent", intentStr).Msg("AI extracted intent")

	// Map the intent string to IntentType
	intent := ic.parseIntent(intentStr)
	if !intent.IsValid() {
		ic.logger.Warn().Str("intent", intentStr).Msg("Unknown intent, defaulting to UNKNOWN_INTENT")
		return domain.IntentUnknown, nil
	}

	return intent, nil
}
func (ic *IntentClassifier) constructPrompt() string {
	// Build intent types description
	var intentDesc strings.Builder
	for _, intent := range domain.AllIntents() {
		intentDesc.WriteString(fmt.Sprintf("* %s\n\n", intent))
	}

	return fmt.Sprintf(`You are an advanced email intent classifier. Your task is to analyze an email and determine the sender's primary intent based on a specific set of intent types and rules.

Here are the available intent types:

<intent_types>
%s
</intent_types>

Instructions:

1. Read the entire email and any associated thread carefully.
2. Analyze the content to identify the sender's primary intent.
3. Use the following classification priority order:
   1. DO_NOT_CONTACT
   2. OUT_OF_OFFICE
   3. REQUEST_FOR_PRICING
   4. REFERRAL
   5. REQUEST_FOR_INFORMATION
   6. NURTURE
   7. INTERESTED
   8. NOT_INTERESTED
   9. UNKNOWN_INTENT

4. Key Rules:
   - Consider multiple intents, but classify based on the strongest signal.
   - Take into account the full context of the email.
   - If the intent is unclear, default to UNKNOWN_INTENT.

5. Edge Cases:
   - Interest vs Nurture:
     * "Check next week" indicates INTERESTED
     * "Next quarter" suggests NURTURE
   - Referral vs Nurture:
     * "New CTO starts soon" implies NURTURE
     * "Contact new CTO" is a REFERRAL
   - Interest vs Information:
     * "How does it work?" is REQUEST_FOR_INFORMATION
     * "Show me how it works" indicates INTERESTED
   - Pricing vs Information:
     * "What's included?" is REQUEST_FOR_INFORMATION
     * "What's the cost?" is REQUEST_FOR_PRICING

6. Analysis Process:
   Before providing your final classification, wrap your analysis inside <intent_analysis> tags. Consider the following:
   - For each intent type, in order of priority:
     * List key phrases or signals that support this intent.
     * List any evidence that contradicts this intent.
     * Evaluate how strongly the email aligns with this intent.
   - Address any potential ambiguities or competing intents.
   - Summarize your final decision and reasoning.

7. Output Format:
   After your analysis, provide the final classification as a single intent label, exactly as it appears in the intent types list. Do not include any explanation with the final output.

Example output structure:

<intent_analysis>
[Your detailed analysis of the email, considering various intents and explaining your reasoning]
</intent_analysis>

<detected_intent>
INTENT_LABEL
</detected_intent>

Remember, your final output must be only the intent label, enclosed in <detected_intent> </detected_intent>, with no additional text.

Now, please analyze the email provided above and classify its intent.`, intentDesc.String())
}

// extractIntent extracts the intent from the XML-like tags in the response
func (ic *IntentClassifier) extractIntent(content string) string {
	// Try to extract from <detected_intent> tags
	re := regexp.MustCompile(`<detected_intent>\s*([A-Z_]+)\s*</detected_intent>`)
	matches := re.FindStringSubmatch(content)

	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	// Fallback: try to find any intent keyword in the content
	contentUpper := strings.ToUpper(content)
	for _, intent := range domain.AllIntents() {
		intentUpper := strings.ToUpper(string(intent))
		intentUpper = strings.ReplaceAll(intentUpper, " ", "_")
		if strings.Contains(contentUpper, intentUpper) {
			return intentUpper
		}
	}

	return "UNKNOWN_INTENT"
}

// parseIntent converts a string to IntentType
func (ic *IntentClassifier) parseIntent(intentStr string) domain.IntentType {
	// Normalize the string
	intentStr = strings.TrimSpace(intentStr)
	intentStr = strings.ToUpper(intentStr)

	// Map intent strings to types
	intentMap := map[string]domain.IntentType{
		"INTERESTED":              domain.IntentInterested,
		"NOT_INTERESTED":          domain.IntentNotInterested,
		"REFERRAL":                domain.IntentReferral,
		"REQUEST_FOR_PRICING":     domain.IntentRequestForPricing,
		"REQUEST_FOR_INFORMATION": domain.IntentRequestForInfo,
		"NURTURE":                 domain.IntentNurture,
		"DO_NOT_CONTACT":          domain.IntentDoNotContact,
		"OUT_OF_OFFICE":           domain.IntentOutOfOffice,
		"UNKNOWN_INTENT":          domain.IntentUnknown,
		"DONOTCONTACT":            domain.IntentDoNotContact,
		"OUTOFOFFICE":             domain.IntentOutOfOffice,
		"REQUESTFORPRICING":       domain.IntentRequestForPricing,
		"REQUESTFORINFORMATION":   domain.IntentRequestForInfo,
		"NOT INTERESTED":          domain.IntentNotInterested,
		"DO NOT CONTACT":          domain.IntentDoNotContact,
		"OUT OF OFFICE":           domain.IntentOutOfOffice,
		"REQUEST FOR PRICING":     domain.IntentRequestForPricing,
		"REQUEST FOR INFORMATION": domain.IntentRequestForInfo,
		"UNKNOWN":                 domain.IntentUnknown,
	}

	if intent, ok := intentMap[intentStr]; ok {
		return intent
	}

	return domain.IntentUnknown
}
