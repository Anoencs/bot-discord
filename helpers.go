package main

func convertToCMCID(geckoID string) string {
	if id, ok := convertToCMCSymbolMap[geckoID]; ok {
		return id
	}
	return geckoID
}

func convertToBinanceSymbol(geckoID string) string {
	if symbol, ok := convertToBinanceSymbolMap[geckoID]; ok {
		return symbol
	}
	return ""
}
