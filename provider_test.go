package mersennet

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const revertHex = "0x08c379a0" +
	"0000000000000000000000000000000000000000000000000000000000000020" +
	"000000000000000000000000000000000000000000000000000000000000000e" +
	"756e6b6e6f776e206d61726b6574000000000000000000000000000000000000"

func node(t *testing.T, body string) *Provider {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return NewProvider(srv.URL)
}

func TestCallRevertKeepsCodeAndData(t *testing.T) {
	p := node(t, `{"jsonrpc":"2.0","id":1,"error":{"code":3,"message":"execution reverted: unknown market","data":"`+revertHex+`"}}`)
	_, err := p.Call(map[string]interface{}{"to": "0x0000000000000000000000000000000000000100", "data": "0x"})
	if err == nil {
		t.Fatal("a revert is an error")
	}
	me, ok := err.(*MersennetError)
	if !ok || me.Code != 3 || me.Message != "execution reverted: unknown market" {
		t.Fatalf("unexpected error %#v", err)
	}
	data, ok := RevertData(err)
	if !ok || data != revertHex {
		t.Fatalf("revert data %q ok=%v", data, ok)
	}
}

func TestCallHaltHasNoRevertData(t *testing.T) {
	p := node(t, `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"out of gas"}}`)
	_, err := p.Call(map[string]interface{}{"to": "0x0000000000000000000000000000000000000100"})
	if err == nil {
		t.Fatal("a halt is an error")
	}
	if _, ok := RevertData(err); ok {
		t.Fatal("a halt carries no revert data")
	}
	if me := err.(*MersennetError); me.Code != -32000 || me.Data != nil {
		t.Fatalf("unexpected error %#v", me)
	}
}

func TestCallSuccessUnchanged(t *testing.T) {
	p := node(t, `{"jsonrpc":"2.0","id":1,"result":"0x2a"}`)
	out, err := p.Call(map[string]interface{}{"to": "0x0000000000000000000000000000000000000100"})
	if err != nil || out != "0x2a" {
		t.Fatalf("out %q err %v", out, err)
	}
}
