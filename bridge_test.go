package holochain

import (
	"encoding/json"
	"fmt"
	. "github.com/holochain/holochain-proto/hash"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestBridgeCall(t *testing.T) {
	d, _, h := PrepareTestChain("test")
	defer CleanupTestChain(h, d)

	token := "bogus token"
	var err error
	Convey("it should fail calls to functions when there's no brided", t, func() {
		_, err = h.BridgeCall("zySampleZome", "testStrFn1", "arg1 arg2", token)
		So(err.Error(), ShouldEqual, "no active bridge")
	})

	fakeFromApp, _ := NewHash("QmVGtdTZdTFaLsaj2RwdVG8jcjNNcp1DE914DKZ2kHmXHx")
	Convey("it should call the bridgeGenesis function when bridging on the to side", t, func() {
		ShouldLog(h.nucleus.alog, func() {
			token, err = h.AddBridgeAsCallee(fakeFromApp, "app data")
			So(err, ShouldBeNil)
		}, `bridge genesis to-- other side is:`+fakeFromApp.String()+` bridging data:app data`)
		c := Capability{Token: token, db: h.bridgeDB}
		bridgeSpecStr, err := c.Validate(nil)
		So(err, ShouldBeNil)
		So(bridgeSpecStr, ShouldEqual, `{"jsSampleZome":{"getProperty":true},"zySampleZome":{"testStrFn1":true}}`)
	})

	fakeToApp, _ := NewHash("QmVGtdTZdTFaLsaj2RwdVG8jcjNNcp1DE914DKZ2kHmXHy")
	url := "http://localhost:31415"
	Convey("it should fail on an unknown bridging zome", t, func() {
		err := h.AddBridgeAsCaller("fooZome", fakeToApp, "fakeAppName", token, url, "app data")
		So(err.Error(), ShouldEqual, "error getting bridging zome: unknown zome: fooZome")
	})

	Convey("it should call the bridgeGenesis function when bridging on the from side", t, func() {

		bridgeZome := h.nucleus.dna.Zomes[0].Name
		ShouldLog(h.nucleus.alog, func() {
			url := "http://localhost:31415"
			err := h.AddBridgeAsCaller(bridgeZome, fakeToApp, "fakeAppName", token, url, "app data")
			So(err, ShouldBeNil)
		}, `bridge genesis from-- other side is:`+fakeToApp.String()+` bridging data:app data`)
	})

	Convey("it should call the bridged function", t, func() {
		var result interface{}
		result, err = h.BridgeCall("zySampleZome", "testStrFn1", "arg1 arg2", token)
		So(err, ShouldBeNil)
		So(result.(string), ShouldEqual, "result: arg1 arg2")
	})

	Convey("it should fail calls to functions not included in the bridge", t, func() {
		_, err = h.BridgeCall("zySampleZome", "testStrFn2", "arg1 arg2", token)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "function not bridged")
	})

}

func TestBridgeSpec(t *testing.T) {
	spec := BridgeSpec{
		"bridgedZome": {"bridgedFunc": true},
	}
	Convey("it should fail functions not in the spec", t, func() {
		So(checkBridgeSpec(spec, "someZome", "someFunc"), ShouldBeFalse)
		So(checkBridgeSpec(spec, "bridgedZome", "someFunc"), ShouldBeFalse)
		So(checkBridgeSpec(spec, "someZome", "bridgedFunc"), ShouldBeFalse)
	})
	Convey("it should not fail functions in the spec", t, func() {
		So(checkBridgeSpec(spec, "bridgedZome", "bridgedFunc"), ShouldBeTrue)
	})
}

func TestBridgeSpecMake(t *testing.T) {
	d, _, h := PrepareTestChain("test")
	defer CleanupTestChain(h, d)

	Convey("it should make spec from the dna", t, func() {
		spec := h.makeBridgeSpec()
		bridgeSpecB, _ := json.Marshal(spec)

		So(fmt.Sprintf("%s", string(bridgeSpecB)), ShouldEqual, `{"jsSampleZome":{"getProperty":true},"zySampleZome":{"testStrFn1":true}}`)
	})
}

func TestBridgeStore(t *testing.T) {
	d, _, h := SetupTestChain("test")
	defer CleanupTestChain(h, d)

	hash, _ := NewHash("QmVGtdTZdTFaLsaj2RwdVG8jcjNNcp1DE914DKZ2kHmXHw")
	token := "some token"
	url := "http://localhost:31415"
	Convey("it should add ba token to the bridged apps list", t, func() {
		err := h.AddBridgeAsCaller("jsSampleZome", hash, "fakeAppName", token, url, "")
		So(err, ShouldBeNil)
		t, u, err := h.GetBridgeToken(hash)
		So(err, ShouldBeNil)
		So(t, ShouldEqual, token)
		So(u, ShouldEqual, url)
	})
}

func TestBridgeGetBridges(t *testing.T) {
	d, _, h := SetupTestChain("test")
	defer CleanupTestChain(h, d)

	Convey("it should return an empty list", t, func() {
		bridges, err := h.GetBridges()
		So(err, ShouldBeNil)
		So(len(bridges), ShouldEqual, 0)
	})

	fakeToApp, _ := NewHash("QmVGtdTZdTFaLsaj2RwdVG8jcjNNcp1DE914DKZ2kHmXHw")
	token := "some token"
	url := "http://localhost:31415"
	err := h.AddBridgeAsCaller("jsSampleZome", fakeToApp, "fakeAppName", token, url, "")
	if err != nil {
		panic(err)
	}

	fakeFromApp, _ := NewHash("QmVGtdTZdTFaLsaj2RwdVG8jcjNNcp1DE914DKZ2kHmXHx")
	_, err = h.AddBridgeAsCallee(fakeFromApp, "app data")
	if err != nil {
		panic(err)
	}

	Convey("it should return the bridged apps", t, func() {
		bridges, err := h.GetBridges()
		So(err, ShouldBeNil)
		So(bridges[0].Side, ShouldEqual, BridgeCaller)
		So(bridges[0].CalleeApp.String(), ShouldEqual, "QmVGtdTZdTFaLsaj2RwdVG8jcjNNcp1DE914DKZ2kHmXHw")
		So(bridges[0].CalleeName, ShouldEqual, "fakeAppName")
		So(bridges[1].Side, ShouldEqual, BridgeCallee)
		So(bridges[1].Token, ShouldNotEqual, 0)
	})
}
