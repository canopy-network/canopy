#!/usr/bin/env python3
"""
Copyright (c) 2026 Sawelew Tech / Ortoplex Research Division

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

NOTE: Portions of this file implement the cross7 / G2 / QMRS algorithms
covered by patent application PATENT_QMRS.md (Zastrzeżenie 1).
See NOTICE for licensing restrictions on cryptographic derivative use.

G2 ZeroPerceptron VM — Canopy L1 plugin (czysty python, bez torch/numpy)
=========================================================================
Wagi: 277 520 floatow wyeksportowane z treningu GPU RTX 4060
(13400+ epok, acc=1.0) — murowana jako zrodlo AI dla cansatell.

Architektura (dokladnie jak trening):
  fc00[256,28] -> block1[512,256] -> block2[256,512] -> head[16,256]

Uzycie przez contract.py:

    from ai_model import G2Model
    model = G2Model()  # lazy - laduje wagi z data/perceptron_genlayer_weights.json
    result = model.predict_from_event(text="...", numbers=[...], ts=1234.0)
    # result: {"y": int, "logits": [...], "decision": "LOW_RISK"|"RAISE_ALERT"|...}
"""

import math
import hashlib
import json
import os
from collections import deque

# ============================================================
# STALE (synchroniczne z colab_perceptron_multimodal.py)
# ============================================================
PHI = (1 + 5 ** 0.5) / 2
GA = 2 * math.pi * (1 - 1 / PHI)

_WEIGHTS_DEFAULT = os.path.join(
    os.path.dirname(os.path.abspath(__file__)),
    "data", "perceptron_genlayer_weights.json",
)

N_CLASSES = 16
N_FEATURES = 28  # extended multimodal 7+14+7


# ============================================================
# 1. ALGEBRA G2 — cross7
# ============================================================
def cross7(a: list, b: list) -> list:
    """Iloczyn oktonionowy 7D (grupa G2)."""
    return [
        a[1]*b[3] - a[3]*b[1] + a[2]*b[6] - a[6]*b[2] + a[4]*b[5] - a[5]*b[4],
        a[2]*b[4] - a[4]*b[2] + a[3]*b[0] - a[0]*b[3] + a[5]*b[6] - a[6]*b[5],
        a[3]*b[5] - a[5]*b[3] + a[4]*b[1] - a[1]*b[4] + a[6]*b[0] - a[0]*b[6],
        a[4]*b[6] - a[6]*b[4] + a[5]*b[2] - a[2]*b[5] + a[0]*b[1] - a[1]*b[0],
        a[5]*b[0] - a[0]*b[5] + a[6]*b[3] - a[3]*b[6] + a[1]*b[2] - a[2]*b[1],
        a[6]*b[1] - a[1]*b[6] + a[0]*b[4] - a[4]*b[0] + a[2]*b[3] - a[3]*b[2],
        a[0]*b[2] - a[2]*b[0] + a[1]*b[5] - a[5]*b[1] + a[3]*b[4] - a[4]*b[3],
    ]


# ============================================================
# 2. ENCODERY 7D — deterministyczne (czysty python)
# ============================================================
class TextEncoder7D:
    DIM = 7

    @staticmethod
    def encode(text: str) -> list:
        v = [0.0] * 7
        if not text:
            return v
        grams = [text[i:i + 3] for i in range(max(1, len(text) - 2))]
        for g in grams:
            h = int(hashlib.md5(g.encode("utf-8")).hexdigest()[:8], 16)
            for k in range(4):
                v[k] += math.sin((h >> (k * 8)) % 256 * GA + k * PHI)
        denom = (len(grams) + 1e-8)
        for k in range(4):
            v[k] /= denom
        counts = {}
        for ch in text:
            counts[ch] = counts.get(ch, 0) + 1
        probs = [c / len(text) for c in counts.values()]
        entropy = -(sum(p * math.log(p + 1e-12) for p in probs)) / 4.0
        v[4] = entropy
        v[5] = sum(c.isdigit() for c in text) / len(text) if text else 0.0
        v[6] = math.log1p(len(text)) / 8.0
        return [max(-1.0, min(1.0, x)) for x in v]


class NumericEncoder7D:
    DIM = 7

    @staticmethod
    def encode(numbers: list) -> list:
        v = [0.0] * 7
        xs = sorted([float(n) for n in numbers if n is not None and n > 0])
        if not xs:
            return v
        mn, mx = min(xs), max(xs)
        for k in range(min(3, len(xs))):
            v[k] = (xs[k] - mn) / (mx - mn + 1e-8)
        if len(xs) >= 2 and xs[1] > 0:
            v[3] = math.log(xs[0] / xs[1] + 1e-8)
        mean = sum(xs) / len(xs)
        v[4] = math.log1p(mean) / 8.0
        var = sum((x - mean) ** 2 for x in xs)
        v[5] = math.sqrt(var / len(xs)) / (mean + 1e-8)
        v[6] = sum(1.0 / x for x in xs)
        return [max(-1.0, min(1.0, x)) for x in v]


class TemporalEncoder:
    DIM = 7

    @staticmethod
    def encode(timestamp: float) -> list:
        import datetime as _dt
        dtm = _dt.datetime.fromtimestamp(timestamp)
        v = [0.0] * 7
        ang_h = 2 * math.pi * (dtm.hour + dtm.minute / 60.0) / 24.0
        v[0], v[1] = math.sin(ang_h), math.cos(ang_h)
        ang_d = 2 * math.pi * dtm.weekday() / 7.0
        v[2], v[3] = math.sin(ang_d), math.cos(ang_d)
        ang_m = 2 * math.pi * (dtm.day - 1) / 30.0
        v[4], v[5] = math.sin(ang_m), math.cos(ang_m)
        v[6] = math.sin(GA * (timestamp / 86400.0))
        return v


# ============================================================
# 3. FUZJA G2 extended 28D
# ============================================================
class G2Encoder:
    def __init__(self, window: int = 30):
        self.window = window
        self._recent = deque(maxlen=window)

    def _raw_average(self, modalities: dict, include: list) -> list:
        acc = [0.0] * 7
        n = 0
        for name in include:
            v = modalities.get(name)
            if v is not None and len(v) == 7:
                acc = [acc[i] + v[i] for i in range(7)]
                n += 1
        if n > 0:
            acc = [x / n for x in acc]
        return acc

    def _octo_fusion(self, modalities: dict) -> list:
        vecs = [v for v in modalities.values()
                if v is not None and hasattr(v, "__len__") and len(v) == 7]
        if not vecs:
            return [0.0] * 14
        if len(vecs) == 1:
            return vecs[0] + [0.0] * 7
        fused = [0.0] * 7
        norms = [0.0] * 7
        n_pairs = 0
        for i in range(len(vecs)):
            for k in range(i + 1, len(vecs)):
                c = cross7(vecs[i], vecs[k])
                fused = [fused[j] + c[j] for j in range(7)]
                norms = [norms[j] + abs(c[j]) for j in range(7)]
                n_pairs += 1
        fused = [x / max(1, n_pairs) for x in fused]
        norms = [x / max(1, n_pairs) for x in norms]
        scale = max(abs(x) for x in fused) + 1e-8
        fused = [x / scale for x in fused]
        return fused + norms

    def encode(self, text: str, numbers: list,
               timestamp: float = 0.0) -> list:
        m_text = TextEncoder7D.encode(text)
        m_num = NumericEncoder7D.encode(numbers)
        m_time = TemporalEncoder.encode(timestamp) if timestamp > 0 else None
        modalities = {"text": m_text, "numbers": m_num}
        present = ["text", "numbers"]
        if m_time is not None:
            modalities["time"] = m_time
            present.append("time")

        fused14 = self._octo_fusion(modalities)
        ctx = (float(sum(self._recent) / len(self._recent))
               if len(self._recent) >= 5 else 0.5)
        fused14 = list(fused14)
        fused14[13] = 0.5 * fused14[13] + 0.5 * ctx
        self._recent.append(fused14[13])

        # extended: raw7 + fused14 + inter7
        raw7 = self._raw_average(modalities, present)
        inter7 = [raw7[i] * fused14[i] for i in range(7)]
        return raw7 + fused14 + inter7

    def reset(self):
        self._recent.clear()


# ============================================================
# 4. BACKBONE — matmul + zero-aktywacja + batchnorm (eval)
# ============================================================
def matmul_vec(w: list, x: list, bias: list) -> list:
    """w: (out,in), x: (in), bias: (out)"""
    return [sum(w[i][j] * x[j] for j in range(len(x))) + bias[i]
            for i in range(len(w))]


def zero_act(s: list) -> list:
    out = []
    for val in s:
        a = 1.0 / (1.0 + abs(val))
        b = val / (1.0 + abs(val))
        out.append(a * b + a * math.tanh(val))
    return out


def batch_norm_eval(x: list, w: list, b: list,
                    rm: list, rv: list, eps: float = 1e-5) -> list:
    out = []
    for i, xi in enumerate(x):
        yi = (xi - rm[i]) / math.sqrt(rv[i] + eps)
        out.append(yi * w[i] + b[i])
    return out


class ZeroPerceptronVM:
    """Czysto-pythonowy forward wg checkpoint (eval mode)."""

    def __init__(self, weights: dict):
        self.w = weights

    def forward(self, x28: list) -> list:
        w = self.w
        # fc00: [256,28]
        s = matmul_vec(w["backbone.fc00.weight"], x28, w["backbone.fc00.bias"])
        h = zero_act(s)

        # block1 512[256 -> 512]
        s = matmul_vec(w["backbone.block1.fc.weight"], h,
                       w["backbone.block1.fc.bias"])
        h = zero_act(s)
        h = batch_norm_eval(h,
                            w["backbone.block1.bn.weight"],
                            w["backbone.block1.bn.bias"],
                            w["backbone.block1.bn.running_mean"],
                            w["backbone.block1.bn.running_var"])

        # block2 256[256,512]
        s = matmul_vec(w["backbone.block2.fc.weight"], h,
                       w["backbone.block2.fc.bias"])
        h = zero_act(s)
        h = batch_norm_eval(h,
                            w["backbone.block2.bn.weight"],
                            w["backbone.block2.bn.bias"],
                            w["backbone.block2.bn.running_mean"],
                            w["backbone.block2.bn.running_var"])

        # head -> 16 logits
        return matmul_vec(w["backbone.head.weight"], h,
                          w["backbone.head.bias"])


# ============================================================
# 5. GŁÓWNY MODEL — ładowanie wag + inferencja
# ============================================================
_CLASS_NAMES = [
    "BENIGN",       # 0
    "LOW_RISK",     # 1
    "INFO",         # 2
    "RAISE_ALERT",  # 3
    "MEME",         # 4
    "SENTIMENT_UP", # 5
    "SENTIMENT_DN", # 6
    "VOLATILITY",   # 7
    "LIQUIDITY",    # 8
    "RESONANCE_UP", # 9
    "RESONANCE_DN", # 10
    "RECURRENCE",   # 11
    "FRACTAL_SELL", # 12
    "FRACTAL_BUY",  # 13
    "AUTO_ONLY",    # 14
    "ANOMALY",      # 15
]


class G2Model:
    """Lazy singleton — wagi ladowane tylko gdy potrzebne."""

    def __init__(self, weights_path: str = None):
        self._weights_bytes = None
        self._model = None
        self._encoder = G2Encoder(window=30)
        self._path = weights_path or _WEIGHTS_DEFAULT

    def _ensure_loaded(self):
        if self._model is not None:
            return
        if not os.path.exists(self._path):
            raise FileNotFoundError(
                "G2Model: brak pliku wag: {}".format(self._path))
        with open(self._path) as f:
            data = json.load(f)
        self._model = ZeroPerceptronVM(data["weights"])
        self.meta = data.get("meta", {})

    def reset(self):
        self._encoder.reset()

    def vectorize(self, text: str, numbers: list,
                  timestamp: float = 0.0) -> list:
        return self._encoder.encode(text, numbers, timestamp)

    def predict_from_event(self, text: str = "", numbers=None,
                           timestamp: float = 0.0) -> dict:
        numbers = numbers if numbers is not None else []
        x = self.vectorize(text, numbers, timestamp)
        return self._predict_impl(x)

    def predict_raw(self, features28: list) -> dict:
        if len(features28) != N_FEATURES:
            raise ValueError(
                "features28 must be exactly {} floats".format(N_FEATURES))
        return self._predict_impl(features28)

    def _predict_impl(self, x28: list, temperature: float = 1.5) -> dict:
        self._ensure_loaded()
        logits = self._model.forward(x28)
        y = logits.index(max(logits))
        probs = self._softmax(logits, temperature)

        # top-3 klasy (nazwy + prawdopodobieństwa)
        ranked = sorted(zip(_CLASS_NAMES, probs), key=lambda kv: kv[1],
                        reverse=True)
        top3 = [{"class": name, "prob": round(float(p), 6)}
                for name, p in ranked[:3]]

        # feature importance — numeryczny gradient logitu klasy zwycięskiej
        importance = self._feature_importance(x28, logits, y)

        return {
            "y": int(y),
            "class": _CLASS_NAMES[y] if 0 <= y < len(_CLASS_NAMES) else "CLASS_{}".format(y),
            "logits": [round(float(v), 6) for v in logits],
            "probs": [round(float(p), 6) for p in probs],
            "top3": top3,
            "feature_importance": [round(float(v), 6) for v in importance],
            "temperature": float(temperature),
            "vector": [round(float(v), 6) for v in x28],
            "meta": self.meta,
        }

    def _feature_importance(self, x28: list, logits: list, y: int,
                            eps: float = 1e-3) -> list:
        """Numeryczny gradient logitu klasy y względem każdego z 28 wymiarów."""
        base = logits[y]
        imp = []
        for i in range(len(x28)):
            x_plus = list(x28)
            x_plus[i] += eps
            logits_plus = self._model.forward(x_plus)
            # normalizacja przez eps — przybliżenie ∂L_y/∂x_i
            imp.append((logits_plus[y] - base) / eps)
        # normalizacja do [-1, 1] dla czytelności
        mx = max(abs(v) for v in imp) if imp else 1.0
        if mx > 1e-12:
            imp = [v / mx for v in imp]
        return imp

    @staticmethod
    def _softmax(logits: list, temperature: float = 1.0) -> list:
        if temperature <= 0:
            temperature = 1.0
        scaled = [v / temperature for v in logits]
        m = max(scaled)
        exp = [math.exp(v - m) for v in scaled]
        s = sum(exp)
        return [e / s for e in exp]


# ============================================================
# 6. SINGLETON (Memoizacja dla hot-loop w contract.deliver_tx)
# ============================================================
_g_model = None


def get_model() -> G2Model:
    global _g_model
    if _g_model is None:
        _g_model = G2Model()
    return _g_model


def reset_model():
    global _g_model
    _g_model = None


if __name__ == "__main__":
    # Lokalny sanity-test: identyczna wejście wejście w/ genlayer_perceptron_vm.py
    m = get_model()
    import sys
    sample = {
        "text": "mecz live bramka",
        "numbers": [1.0, 2.0, 3.0],
        "timestamp": 1234567890.0,
    }
    res = m.predict_from_event(**sample)
    print(json.dumps({
        "y": res["y"],
        "class": res["class"],
        "top3": res["logits"][:3],
        "vector_len": len(res["vector"]),
    }, indent=2))