"""CI で本体の代わりをする小さな道具(取り決めは PROTOCOL.md)。

インストールの手順を通しで試すとき、pair の待ちを終わらせたり、
操作を実際に動かしたりするのに使う。標準ライブラリだけで書き、
Windows・Mac・Linux の CI でそのまま動くようにしている。

  python3 ci/hub_stub.py pair      組める時間が開くのを待って鍵を渡す
  python3 ci/hub_stub.py list      一覧を取り、署名を確かめて表示する
  python3 ci/hub_stub.py run <id>  操作を動かす
"""
import hashlib
import hmac
import json
import sys
import time
import urllib.error
import urllib.request

BASE = "http://127.0.0.1:47821"
KEY = bytes(range(32))  # CI だけで使う鍵。秘密ではない


def sign(*parts: bytes) -> str:
    return hmac.new(KEY, b"\n".join(parts), hashlib.sha256).hexdigest()


def request(path, body=None):
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(BASE + path, data=data, method="GET" if data is None else "POST")
    try:
        with urllib.request.urlopen(req, timeout=3) as res:
            return res.status, res.read(), res.headers.get("X-Kido-Sig")
    except urllib.error.HTTPError as e:
        return e.code, e.read(), None


def hello():
    status, raw, _ = request("/v1/hello")
    assert status == 200, raw
    return json.loads(raw)


def pair():
    deadline = time.time() + 120
    while time.time() < deadline:
        try:
            if hello()["state"] == "pairing":
                status, raw, _ = request("/v1/pair", {"key": KEY.hex()})
                print("pair:", status, raw.decode())
                return 0 if status == 200 else 1
        except (OSError, AssertionError):
            pass  # まだ起動していない
        time.sleep(1)
    print("pair: 組める時間が開きませんでした")
    return 1


def call(kind, extra):
    nonce = hello()["nonce"]
    parts = [kind.encode(), nonce.encode()] + [e.encode() for e in extra.values()]
    body = {"nonce": nonce, **extra, "sig": sign(*parts)}
    status, raw, sig = request("/v1/" + kind, body)
    print(kind + ":", status, raw.decode())
    if status != 200:
        return 1
    if sig != sign((kind + "-ok").encode(), nonce.encode(), raw):
        print("署名が合いません")
        return 1
    return 0


def main(argv):
    if argv[:1] == ["pair"]:
        return pair()
    if argv[:1] == ["list"]:
        return call("list", {})
    if argv[:1] == ["run"] and len(argv) == 2:
        return call("run", {"id": argv[1]})
    print(__doc__)
    return 2


if __name__ == "__main__":
    sys.stdout.reconfigure(encoding="utf-8")
    sys.exit(main(sys.argv[1:]))
