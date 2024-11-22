package main

type CryptoInfo struct {
	Symbol  string
	GeckoID string
}

var commonCryptos = map[string]CryptoInfo{
	"bitcoin":       {Symbol: "BTC", GeckoID: "bitcoin"},
	"ethereum":      {Symbol: "ETH", GeckoID: "ethereum"},
	"binancecoin":   {Symbol: "BNB", GeckoID: "binancecoin"},
	"ripple":        {Symbol: "XRP", GeckoID: "ripple"},
	"cardano":       {Symbol: "ADA", GeckoID: "cardano"},
	"solana":        {Symbol: "SOL", GeckoID: "solana"},
	"dogecoin":      {Symbol: "DOGE", GeckoID: "dogecoin"},
	"polkadot":      {Symbol: "DOT", GeckoID: "polkadot"},
	"matic-network": {Symbol: "MATIC", GeckoID: "matic-network"},
	"avalanche-2":   {Symbol: "AVAX", GeckoID: "avalanche-2"},
	"worldcoin-wld": {Symbol: "WLD", GeckoID: "worldcoin-wld"},
	"avail":         {Symbol: "AVAIL", GeckoID: "avail"},
	"starknet":      {Symbol: "STRK", GeckoID: "starknet"},
	// Additional top cryptocurrencies
	"tron":                      {Symbol: "TRX", GeckoID: "tron"},
	"chainlink":                 {Symbol: "LINK", GeckoID: "chainlink"},
	"uniswap":                   {Symbol: "UNI", GeckoID: "uniswap"},
	"litecoin":                  {Symbol: "LTC", GeckoID: "litecoin"},
	"bitcoin-cash":              {Symbol: "BCH", GeckoID: "bitcoin-cash"},
	"stellar":                   {Symbol: "XLM", GeckoID: "stellar"},
	"monero":                    {Symbol: "XMR", GeckoID: "monero"},
	"cosmos":                    {Symbol: "ATOM", GeckoID: "cosmos"},
	"ethereum-classic":          {Symbol: "ETC", GeckoID: "ethereum-classic"},
	"filecoin":                  {Symbol: "FIL", GeckoID: "filecoin"},
	"hedera-hashgraph":          {Symbol: "HBAR", GeckoID: "hedera-hashgraph"},
	"near":                      {Symbol: "NEAR", GeckoID: "near"},
	"vechain":                   {Symbol: "VET", GeckoID: "vechain"},
	"quant-network":             {Symbol: "QNT", GeckoID: "quant-network"},
	"algorand":                  {Symbol: "ALGO", GeckoID: "algorand"},
	"eos":                       {Symbol: "EOS", GeckoID: "eos"},
	"decentraland":              {Symbol: "MANA", GeckoID: "decentraland"},
	"the-sandbox":               {Symbol: "SAND", GeckoID: "the-sandbox"},
	"axie-infinity":             {Symbol: "AXS", GeckoID: "axie-infinity"},
	"aave":                      {Symbol: "AAVE", GeckoID: "aave"},
	"tezos":                     {Symbol: "XTZ", GeckoID: "tezos"},
	"theta-token":               {Symbol: "THETA", GeckoID: "theta-token"},
	"fantom":                    {Symbol: "FTM", GeckoID: "fantom"},
	"kucoin-shares":             {Symbol: "KCS", GeckoID: "kucoin-shares"},
	"gatetoken":                 {Symbol: "GT", GeckoID: "gatetoken"},
	"neo":                       {Symbol: "NEO", GeckoID: "neo"},
	"maker":                     {Symbol: "MKR", GeckoID: "maker"},
	"curve-dao-token":           {Symbol: "CRV", GeckoID: "curve-dao-token"},
	"compound-governance-token": {Symbol: "COMP", GeckoID: "compound-governance-token"},
	"dash":                      {Symbol: "DASH", GeckoID: "dash"},
	"zcash":                     {Symbol: "ZEC", GeckoID: "zcash"},
	"waves":                     {Symbol: "WAVES", GeckoID: "waves"},
	"kusama":                    {Symbol: "KSM", GeckoID: "kusama"},
	"elrond-erd-2":              {Symbol: "EGLD", GeckoID: "elrond-erd-2"},
	"pancakeswap-token":         {Symbol: "CAKE", GeckoID: "pancakeswap-token"},
	"sushi":                     {Symbol: "SUSHI", GeckoID: "sushi"},
	"yearn-finance":             {Symbol: "YFI", GeckoID: "yearn-finance"},
	"1inch":                     {Symbol: "1INCH", GeckoID: "1inch"},
	"enjincoin":                 {Symbol: "ENJ", GeckoID: "enjincoin"},
	"basic-attention-token":     {Symbol: "BAT", GeckoID: "basic-attention-token"},
	"zilliqa":                   {Symbol: "ZIL", GeckoID: "zilliqa"},
	"harmony":                   {Symbol: "ONE", GeckoID: "harmony"},
	"flow":                      {Symbol: "FLOW", GeckoID: "flow"},
	"celo":                      {Symbol: "CELO", GeckoID: "celo"},
	"rocket-pool":               {Symbol: "RPL", GeckoID: "rocket-pool"},
	"arweave":                   {Symbol: "AR", GeckoID: "arweave"},
	"osmosis":                   {Symbol: "OSMO", GeckoID: "osmosis"},
	"immutable-x":               {Symbol: "IMX", GeckoID: "immutable-x"},
	"kava":                      {Symbol: "KAVA", GeckoID: "kava"},
	"icon":                      {Symbol: "ICX", GeckoID: "icon"},
	"gnosis":                    {Symbol: "GNO", GeckoID: "gnosis"},
	"thorchain":                 {Symbol: "RUNE", GeckoID: "thorchain"},
	"render-token":              {Symbol: "RNDR", GeckoID: "render-token"},
	"xdc-network":               {Symbol: "XDC", GeckoID: "xdc-network"},
	"optimism":                  {Symbol: "OP", GeckoID: "optimism"},
	"arbitrum":                  {Symbol: "ARB", GeckoID: "arbitrum"},
	"sei-network":               {Symbol: "SEI", GeckoID: "sei-network"},
	"celestia":                  {Symbol: "TIA", GeckoID: "celestia"},
	"bonk":                      {Symbol: "BONK", GeckoID: "bonk"},
	"injective-protocol":        {Symbol: "INJ", GeckoID: "injective-protocol"},
	"stacks":                    {Symbol: "STX", GeckoID: "stacks"},
	"sui":                       {Symbol: "SUI", GeckoID: "sui"},
	"pepe":                      {Symbol: "PEPE", GeckoID: "pepe"},
	"fetch-ai":                  {Symbol: "FET", GeckoID: "fetch-ai"},
	"conflux-token":             {Symbol: "CFX", GeckoID: "conflux-token"},
	"mina-protocol":             {Symbol: "MINA", GeckoID: "mina-protocol"},
	"casper-network":            {Symbol: "CSPR", GeckoID: "casper-network"},
	"oasis-network":             {Symbol: "ROSE", GeckoID: "oasis-network"},
	"api3":                      {Symbol: "API3", GeckoID: "api3"},
	"gmx":                       {Symbol: "GMX", GeckoID: "gmx"},
	"pendle":                    {Symbol: "PENDLE", GeckoID: "pendle"},
	"radix":                     {Symbol: "XRD", GeckoID: "radix"},
	"dydx":                      {Symbol: "DYDX", GeckoID: "dydx"},
	"stepn":                     {Symbol: "GMT", GeckoID: "stepn"},
	"blur":                      {Symbol: "BLUR", GeckoID: "blur"},
	"flare-networks":            {Symbol: "FLR", GeckoID: "flare-networks"},
	"polymath":                  {Symbol: "POLY", GeckoID: "polymath"},
	"woo-network":               {Symbol: "WOO", GeckoID: "woo-network"},
	"akash-network":             {Symbol: "AKT", GeckoID: "akash-network"},
	"band-protocol":             {Symbol: "BAND", GeckoID: "band-protocol"},
	"kadena":                    {Symbol: "KDA", GeckoID: "kadena"},
	"orbs":                      {Symbol: "ORBS", GeckoID: "orbs"},
	"mask-network":              {Symbol: "MASK", GeckoID: "mask-network"},
	"nervos-network":            {Symbol: "CKB", GeckoID: "nervos-network"},
	"safemoon":                  {Symbol: "SFM", GeckoID: "safemoon"},
	"vethor-token":              {Symbol: "VTHO", GeckoID: "vethor-token"},
	"mdex":                      {Symbol: "MDX", GeckoID: "mdex"},
	"stargate-finance":          {Symbol: "STG", GeckoID: "stargate-finance"},
	"jupiter":                   {Symbol: "JUP", GeckoID: "jupiter"},
	"raydium":                   {Symbol: "RAY", GeckoID: "raydium"},
	"energy-web-token":          {Symbol: "EWT", GeckoID: "energy-web-token"},

	// DeFi tokens
	"compound":              {Symbol: "COMP", GeckoID: "compound-governance-token"},
	"synthetix":             {Symbol: "SNX", GeckoID: "synthetix-network-token"},
	"balancer":              {Symbol: "BAL", GeckoID: "balancer"},
	"convex-finance":        {Symbol: "CVX", GeckoID: "convex-finance"},
	"olympus":               {Symbol: "OHM", GeckoID: "olympus"},
	"perp-protocol":         {Symbol: "PERP", GeckoID: "perpetual-protocol"},
	"loopring":              {Symbol: "LRC", GeckoID: "loopring"},
	"kyber-network-crystal": {Symbol: "KNC", GeckoID: "kyber-network-crystal"},

	// Memecoins
	"shiba-inu":      {Symbol: "SHIB", GeckoID: "shiba-inu"},
	"floki":          {Symbol: "FLOKI", GeckoID: "floki"},
	"dogelon-mars":   {Symbol: "ELON", GeckoID: "dogelon-mars"},
	"baby-doge-coin": {Symbol: "BABYDOGE", GeckoID: "baby-doge-coin"},

	// Gaming & Metaverse
	"gala":              {Symbol: "GALA", GeckoID: "gala"},
	"illuvium":          {Symbol: "ILV", GeckoID: "illuvium"},
	"magic":             {Symbol: "MAGIC", GeckoID: "magic"},
	"star-atlas":        {Symbol: "ATLAS", GeckoID: "star-atlas"},
	"mines-of-dalarnia": {Symbol: "DAR", GeckoID: "mines-of-dalarnia"},

	// Layer 2 Solutions
	"metis-token": {Symbol: "METIS", GeckoID: "metis-token"},
	"zkspace":     {Symbol: "ZKS", GeckoID: "zkspace"},

	// Infrastructure
	"the-graph":    {Symbol: "GRT", GeckoID: "the-graph"},
	"cartesi":      {Symbol: "CTSI", GeckoID: "cartesi"},
	"ankr":         {Symbol: "ANKR", GeckoID: "ankr"},
	"keep-network": {Symbol: "KEEP", GeckoID: "keep-network"},

	// Privacy Coins
	"secret":  {Symbol: "SCRT", GeckoID: "secret"},
	"horizen": {Symbol: "ZEN", GeckoID: "horizen"},
	"beam":    {Symbol: "BEAM", GeckoID: "beam"},

	// Exchange Tokens
	"ftx-token":   {Symbol: "FTT", GeckoID: "ftx-token"},
	"huobi-token": {Symbol: "HT", GeckoID: "huobi-token"},
	"wazirx":      {Symbol: "WRX", GeckoID: "wazirx"},
	"mx-token":    {Symbol: "MX", GeckoID: "mx-token"},
}

var convertToBinanceSymbolMap = map[string]string{
	"bitcoin":          "BTCUSDT",
	"ethereum":         "ETHUSDT",
	"binancecoin":      "BNBUSDT",
	"ripple":           "XRPUSDT",
	"cardano":          "ADAUSDT",
	"solana":           "SOLUSDT",
	"dogecoin":         "DOGEUSDT",
	"polkadot":         "DOTUSDT",
	"matic-network":    "MATICUSDT",
	"avalanche-2":      "AVAXUSDT",
	"worldcoin-wld":    "WLDUSDT",
	"tron":             "TRXUSDT",
	"chainlink":        "LINKUSDT",
	"uniswap":          "UNIUSDT",
	"litecoin":         "LTCUSDT",
	"stellar":          "XLMUSDT",
	"filecoin":         "FILUSDT",
	"hedera-hashgraph": "HBARUSDT",
	"near":             "NEARUSDT",
	"vechain":          "VETUSDT",
	"algorand":         "ALGOUSDT",
	"tezos":            "XTZUSDT",
	"fantom":           "FTMUSDT",
	"apecoin":          "APEUSDT",
	"shiba-inu":        "SHIBUSDT",
	"the-sandbox":      "SANDUSDT",
	"decentraland":     "MANAUSDT",
	"theta-token":      "THETAUSDT",
	"strk":             "STRKUSDT",
}

var convertToCMCSymbolMap = map[string]string{
	"bitcoin":          "bitcoin",
	"ethereum":         "ethereum",
	"binancecoin":      "binance-coin",
	"ripple":           "ripple",
	"cardano":          "cardano",
	"solana":           "solana",
	"dogecoin":         "dogecoin",
	"polkadot":         "polkadot",
	"matic-network":    "polygon",
	"avalanche-2":      "avalanche",
	"worldcoin-wld":    "worldcoin",
	"tron":             "tron",
	"chainlink":        "chainlink",
	"uniswap":          "uniswap",
	"litecoin":         "litecoin",
	"stellar":          "stellar",
	"filecoin":         "filecoin",
	"hedera-hashgraph": "hedera",
	"near":             "near-protocol",
	"vechain":          "vechain",
	"algorand":         "algorand",
	"tezos":            "tezos",
	"fantom":           "fantom",
	"apecoin":          "apecoin",
	"shiba-inu":        "shiba-inu",
	"the-sandbox":      "the-sandbox",
	"decentraland":     "decentraland",
	"theta-token":      "theta",
	"avail":            "avail",
	"strk":             "strk",
}
