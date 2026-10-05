package seed

import (
	"fmt"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// photos maps each seed product to three Unsplash photo ids (images.unsplash.com/photo-<id>).
// Free Unsplash License: https://unsplash.com/license
var photos = map[string][3]string{
	"Aero Buds Pro":       {"1572569511254-d8f925fe2cbb", "1606841837239-c5a1a4a07af7", "1606741965326-cb990ae01bb2"},
	"Orbit Speaker Mini":  {"1608043152269-423dbba4e7e1", "1589256469067-ea99122bbdc4", "1589003077984-894e133dabab"},
	"Bass Dome 360":       {"1519558260268-cde7e03a0152", "1529359744902-86b2ab9edaea", "1558089687-f282ffcbc126"},
	"Studio Cans X":       {"1505740420928-5e560c06d30e", "1618366712010-f4ae9c647dcb", "1546435770-a3e426bf472b"},
	"Echo Bar Slim":       {"1557376382-e96b6778ffdc", "1779905128767-25107ff9b7d3", "1549199224-29cf2f784cd0"},
	"Aero Buds Lite":      {"1615281612781-4b972bd4e3fe", "1783890848500-426732ee5a11", "1777216794546-35784e0dec81"},
	"Pulse Watch S2":      {"1579586337278-3befd40fd17a", "1660844817855-3ecc7ef21f12", "1546868871-7041f2a55e12"},
	"Halo Ring Gen 3":     {"1651752090085-50375d90bf8b", "1744697307482-0f55e2e0c1b6", "1760088348194-a5ac70a8aa9f"},
	"Stride Band 4":       {"1576243345690-4e4b79b63288", "1434494878577-86c23bcb06b9", "1532435109783-fdb8a2be0baa"},
	"Vital Clip":          {"1508685096489-7aacd43bd3b1", "1575311373937-040b8e1fd5b6", "1696688713460-de12ac76ebc6"},
	"Pulse Watch Ultra":   {"1637160151663-a410315e4e75", "1434493789847-2f02dc6ca35d", "1632794716789-42d9995fb5b6"},
	"Lumen Desk Lamp":     {"1519219788971-8d9797e0928e", "1621447980929-6638614633c8", "1570974802254-4b0ad1a755f5"},
	"Arc Floor Light":     {"1507473885765-e6ed057f782c", "1642689703534-e41f29622078", "1673939859210-23d8444237ff"},
	"Glow Strip 5m":       {"1618403323851-ac3d38029495", "1659066019874-7a15628cea60", "1572249930263-64fc5bbdb14b"},
	`Halo Ring Light 18"`: {"1478826160983-e6db8c7d537a", "1673196649671-eb09066ad6c1", "1598358532244-6480b5c5ea1a"},
	"Ember Bulb E27":      {"1529310399831-ed472b81d589", "1552862750-746b8f6f7f25", "1573621622238-f7ac6ac0429a"},
	"Nimbus Hub":          {"1545259741-2ea3ebf61fa3", "1730967844913-29eb5cae5f34", "1703935932245-0fc22c9e52d7"},
	"Thermo Dot":          {"1545259742-b4fd8fea67e4", "1770625467384-304e461ef1be", "1774876549246-dfa8eae5d9f5"},
	"Sentry Cam 2K":       {"1618482914248-29272d021005", "1549109926-58f039549485", "1496368077930-c1e31b4e5b44"},
	"Mist Diffuser":       {"1634681896994-0027a701b1d7", "1732229035217-e7e42f61af4b", "1787074628644-692102b4090f"},
	"Air Purifier One":    {"1632928274371-878938e4d825", "1709745634912-2a79b938f3c2", "1730299789489-b55bf96b22bf"},
	"Smart Plug Duo":      {"1610056494052-6a4f83a8368c", "1565049981953-379c9c2a5d48", "1610056494071-9373f12bf769"},
	"Flux Keyboard 75":    {"1618384887929-16ec33fab9ef", "1547394765-185e1e68f34e", "1635987391914-cb84b567e68f"},
	"Drift Mouse":         {"1615663245857-ac93bb7c39e7", "1527864550417-7fd91fc51a46", "1605773527852-c546a8584ea3"},
	"Dock Prime 12-in-1":  {"1760376789487-994070337c76", "1616578273461-3a99ce422de6", "1760376789478-c1023d2dc007"},
	"Vertex Monitor Arm":  {"1587831990711-23ca6441447b", "1510519138101-570d1dca3d66", "1575318634028-6a0cfcb60c59"},
	"Nova SSD 2TB":        {"1779896412280-cff477924a69", "1721333088976-33173bdc59a2", "1601737487795-dab272f52420"},
	"Flux Keyboard TKL":   {"1632079003110-d694908500da", "1595044426077-d36d9236d54a", "1601445638532-3c6f6c3aa1d6"},
	"Signal Webcam 4K":    {"1726127461372-547b9ffa4236", "1629429407756-446d66f5b24e", "1715869618915-a7bf6608d4c3"},
	"Atlas Backpack":      {"1553062407-98eeb64c6a62", "1622560480654-d96214fdc887", "1622560480605-d83c853bc5c3"},
	"Vapor Bottle 750ml":  {"1602143407151-7111542de6e8", "1616118132534-381148898bb4", "1544003484-3cd181d17917"},
	"Cable Kit Braided":   {"1572721546624-05bf65ad7679", "1492107376256-4026437926cd", "1595756630452-736bc8ef3693"},
	"Mag Wallet":          {"1614260938313-a7fc1a7ad0d2", "1628483211662-9bcc692c46dc", "1579014134953-1580d7f123f3"},
	`Folio Sleeve 14"`:    {"1689757855413-9e366c2011f1", "1675668409245-955188b96bf6", "1763034179057-acad3a072568"},
	"Grip Stand":          {"1680007889408-4d655577346b", "1680007892800-4ce496b18860", "1676300463288-0a1183aef2fa"},
	"Travel Pouch":        {"1516765865430-ac8d320b9208", "1613896640137-bb5b31496315", "1758798689719-5b554ac3b65a"},
}

func photoURL(id string) string {
	return "https://images.unsplash.com/photo-" + id + "?w=800&h=800&fit=crop&auto=format&q=80"
}

// SeedImage is a stock photo assigned to the gallery slot gen-<product>-<pos>.
type SeedImage struct {
	ProductName string
	Position    int
	URL         string
	Alt         string
}

// SeedImages lists every stock photo, so existing databases can replace the
// generated SVG artwork they were seeded with.
func SeedImages() []SeedImage {
	var out []SeedImage
	for name, ids := range photos {
		for i, id := range ids {
			out = append(out, SeedImage{ProductName: name, Position: i, URL: photoURL(id), Alt: photoAlt(name, i)})
		}
	}
	return out
}

func photoAlt(name string, i int) string { return fmt.Sprintf("%s — photo %d (Unsplash)", name, i+1) }

// seedImages returns the product's Unsplash photos, or generated artwork for
// products without any.
func seedImages(p d.Product) []d.ProductImage {
	ids, ok := photos[p.Name]
	if !ok {
		return generatedImages(p)
	}
	imgs := make([]d.ProductImage, len(ids))
	for i, id := range ids {
		imgs[i] = d.ProductImage{
			ID:        fmt.Sprintf("gen-%d-%d", p.ID, i),
			URL:       photoURL(id),
			Alt:       photoAlt(p.Name, i),
			Generated: true,
		}
	}
	return imgs
}
