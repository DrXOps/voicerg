#!/usr/bin/env python3
"""
analyze.py — консольный генератор отчёта из logs/events.jsonl.

Использование:
    python3 scripts/analyze.py --since=7d [--events=logs/events.jsonl]

Читает events.jsonl (JSON Lines), агрегирует research_complete / stt_response /
llm_response / intent_response / hallucination_check / correction_cycle и печатает
сводку в Markdown (для вставки в отчёт или анализа через Claude Code).

Пример: расход времени куда ушёл, топ медленных исследований, распределение
по профилям, ошибки распознавания, доля корректировок.
"""
import argparse
import json
import statistics
import sys
from collections import defaultdict
from datetime import datetime, timedelta, timezone


def parse_ts(ts: str) -> datetime:
    try:
        return datetime.fromisoformat(ts)
    except Exception:
        return datetime.now(timezone.utc)


def load_events(path: str, since_days: int):
    events = []
    cutoff = datetime.now(timezone.utc) - timedelta(days=since_days)
    try:
        fh = open(path, "r", encoding="utf-8")
    except FileNotFoundError:
        print(f"Файл не найден: {path}", file=sys.stderr)
        return events
    with fh:
        for line in fh:
            line = line.strip()
            if not line:
                continue
            try:
                rec = json.loads(line)
            except json.JSONDecodeError:
                continue
            ts = parse_ts(rec.get("ts", ""))
            if ts < cutoff:
                continue
            events.append(rec)
    return events


def pct(lst, p):
    if not lst:
        return 0
    s = sorted(lst)
    k = (len(s) - 1) * p
    f = int(k)
    c = f + 1
    if c >= len(s):
        return s[-1]
    return s[f] + (s[c] - s[f]) * (k - f)


def md_row(*cells):
    return "| " + " | ".join(str(c) for c in cells) + " |"


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--events", default="logs/events.jsonl")
    ap.add_argument("--since", default="7d", help="1d | 7d | 30d | all")
    args = ap.parse_args()

    days = {"1d": 1, "7d": 7, "30d": 30}.get(args.since, 7)
    if args.since == "all":
        days = 365 * 10
    events = load_events(args.events, days)
    if not events:
        print("Нет событий за выбранный период.")
        return

    researches = []
    by_profile = defaultdict(int)
    stt_lat = []
    llm_lat = []
    intent_lat = []
    wake_lat = []
    paste_lat = []
    hot_total = 0
    halluc_warn = 0
    corrections = 0
    errors = 0

    for e in events:
        ev = e.get("event")
        d = e.get("data") or {}
        if ev == "research_complete":
            researches.append(d)
            by_profile[d.get("profile", "default")] += 1
            tl = d.get("state_timings") or {}
            hot_total += d.get("hotphrases_count", 0)
            halluc_warn += d.get("hallucinations_warned_count", 0)
            corrections += d.get("corrections_count", 0)
            if d.get("llm_latency_ms"): llm_lat.append(d["llm_latency_ms"])
            if d.get("stt_latency_ms"): stt_lat.append(d["stt_latency_ms"])
            if d.get("total_ms"): paste_lat.append(int(d["total_ms"]))
        elif ev == "stt_response":
            if d.get("latency_ms"): stt_lat.append(d["latency_ms"])
            if d.get("error"): errors += 1
        elif ev == "llm_response":
            if d.get("latency_ms"): llm_lat.append(d["latency_ms"])
            if d.get("error"): errors += 1
        elif ev == "intent_response":
            if d.get("latency_ms"): intent_lat.append(d["latency_ms"])
            if d.get("error"): errors += 1
        elif ev == "wake_detected":
            if d.get("detection_latency_ms"): wake_lat.append(d["detection_latency_ms"])
        elif ev == "smart_paste_result":
            if d.get("latency_ms"): paste_lat.append(d["latency_ms"])
            if d.get("error"): errors += 1
        elif ev == "hotphrase_substituted":
            hot_total += 1
        elif ev == "hallucination_check":
            if d.get("warning_shown"): halluc_warn += 1
        elif ev == "correction_cycle":
            corrections += 1
        elif ev == "error":
            errors += 1

    total_ms = [d.get("total_ms", 0) for d in researches if d.get("total_ms")]
    print("# Отчёт по событиям (автоматический анализ)")
    print(f"\n**Период:** {args.since} | **Событий:** {len(events)} | **Исследований:** {len(researches)}")
    print()
    if researches:
        print("## Распределение по профилям")
        print(md_row("Профиль", "Кол-во"))
        for k, v in sorted(by_profile.items(), key=lambda x: -x[1]):
            print(md_row(k, v))
        print()
        print("## Исследования: длительность (total_ms)")
        print(md_row("Метрика", "Значение"))
        if total_ms:
            print(md_row("avg", f"{statistics.mean(total_ms)/1000:.1f}s"))
            print(md_row("p50", f"{pct(total_ms,0.5)/1000:.1f}s"))
            print(md_row("p95", f"{pct(total_ms,0.95)/1000:.1f}s"))
        print()
        top5 = sorted(researches, key=lambda r: r.get("total_ms", 0), reverse=True)[:5]
        if top5:
            print("## Топ-5 самых медленных исследований")
            print(md_row("research_id", "total_ms", "profile"))
            for r in top5:
                print(md_row(r.get("research_id", "?"), r.get("total_ms", 0), r.get("profile", "-")))
            print()

    def lat_block(name, lst):
        print(f"## {name}")
        print(md_row("p50", f"{pct(lst,0.5):.0f}ms"))
        print(md_row("p95", f"{pct(lst,0.95):.0f}ms"))
        print()

    if stt_lat:  lat_block("STT латентность", stt_lat)
    if llm_lat:  lat_block("LLM латентность", llm_lat)
    if intent_lat: lat_block("Intent латентность", intent_lat)
    if wake_lat: lat_block("Wake-detect латентность", wake_lat)
    if paste_lat: lat_block("Вставка латентность", paste_lat)

    print("## Качество")
    print(md_row("Метрика", "Значение"))
    print(md_row("Горячих фраз заменено", hot_total))
    print(md_row("Предупреждения галлюцинаций", halluc_warn))
    print(md_row("Корректировки", corrections))
    print(md_row("Ошибки", errors))
    if researches:
        rate = (corrections / len(researches)) * 100 if researches else 0
        print(md_row("Доля корректировок, %", f"{rate:.1f}"))
    print()


if __name__ == "__main__":
    main()