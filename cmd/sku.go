package cmd

var skuMap = map[GPU]string{
	{Brand: brands.nvidia, Model: "3090"}:                      "6429434",
	{Brand: brands.nvidia, Model: "3080"}:                      "6429440",
	{Brand: brands.evga, Model: "3080", Version: "xc3-ultra"}:  "6432400",
	{Brand: brands.evga, Model: "3080", Version: "ftw3-ultra"}: "6436196",
	{Brand: brands.evga, Model: "3080", Version: "xc3-black"}:  "6432399",
	{Brand: brands.evga, Model: "3080", Version: "ftw3"}:       "6436191",
	{Brand: brands.evga, Model: "3080", Version: "xc3"}:        "6436194",
	{Brand: brands.pny, Model: "3080", Version: "1"}:           "6432655", // as far as I can tell these are
	{Brand: brands.pny, Model: "3080", Version: "2"}:           "6432658", // two skus for the same thing
	{Brand: brands.msi, Model: "3080"}:                         "6430175",
	{Brand: brands.nvidia, Model: "3070"}:                      "6429442",
	{Brand: brands.gigabyte, Model: "3070"}:                    "6437912",
	{Brand: brands.evga, Model: "3070"}:                        "6439300",
	{Brand: brands.pny, Model: "3070"}:                         "6432654",
	{Brand: brands.nvidia, Model: "3060ti"}:                    "6439402",
	{Brand: brands.evga, Model: "3060ti", Version: "ftw3"}:     "6444444",
	{Brand: brands.evga, Model: "3060ti", Version: "xc"}:       "6444445",
	{Brand: brands.asus, Model: "3060ti"}:                      "6452573",
}
