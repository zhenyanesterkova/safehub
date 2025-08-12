package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/zhenyanesterkova/safehub/internal/models"
)

func validateDataContent(dataType models.DataType, data []byte) error {
	switch dataType {
	case models.DataTypeCredentials:
		var creds models.Credentials
		if err := json.Unmarshal(data, &creds); err != nil {
			return fmt.Errorf("invalid credentials format: %w", err)
		}
		if creds.Login == "" || creds.Password == "" {
			return fmt.Errorf("login and password are required")
		}

	case models.DataTypeText:
		var textData models.TextData
		if err := json.Unmarshal(data, &textData); err != nil {
			return fmt.Errorf("invalid text data format: %w", err)
		}

	case models.DataTypeBinary:
		var binaryData models.BinaryData
		if err := json.Unmarshal(data, &binaryData); err != nil {
			return fmt.Errorf("invalid binary data format: %w", err)
		}
		if len(binaryData.Content) == 0 {
			return fmt.Errorf("binary content cannot be empty")
		}

	case models.DataTypeCard:
		var cardData models.CardData
		if err := json.Unmarshal(data, &cardData); err != nil {
			return fmt.Errorf("invalid card data format: %w", err)
		}
		if cardData.Number == "" || cardData.Holder == "" {
			return fmt.Errorf("card number and holder are required")
		}
	}

	return nil
}
