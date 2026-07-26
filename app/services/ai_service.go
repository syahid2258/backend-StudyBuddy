package services

// AIService handles AI chat interactions.
// Currently returns placeholder responses; ready for real AI integration.
type AIService struct{}

func NewAIService() *AIService {
	return &AIService{}
}

// Chat processes a user message and returns an AI response.
// TODO: Replace the placeholder with actual AI provider integration
// (e.g., Gemini, OpenAI) by injecting API key from config.
func (s *AIService) Chat(userID uint, message string) (string, error) {
	// Placeholder — will be replaced with actual AI call in next phase
	reply := "Halo! Saya adalah StudyBuddy AI. Saat ini saya sedang dalam tahap integrasi. " +
		"Pertanyaan Anda telah diterima dan akan segera diproses. " +
		"Silakan tunggu pembaruan berikutnya untuk respons AI yang sesungguhnya. 🤖"
	return reply, nil
}

// ChatWithContext sends a message with topic context.
// TODO: Use projectID to load module content as context for the AI.
func (s *AIService) ChatWithContext(userID uint, projectID uint, message string) (string, error) {
	return s.Chat(userID, message)
}
