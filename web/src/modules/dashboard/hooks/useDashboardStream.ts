import { useEffect, useEffectEvent, useRef, useState } from "react";
import type { DashboardSnapshot } from "../types";

export type DashboardStreamState = "connecting" | "live" | "retrying";

export function useDashboardStream() {
  const mountedRef = useRef(true);
  const streamRef = useRef<EventSource | null>(null);
  const hasDataRef = useRef(false);
  const [data, setData] = useState<DashboardSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [streamState, setStreamState] = useState<DashboardStreamState>("connecting");
  const [streamVersion, setStreamVersion] = useState(0);
  const [error, setError] = useState<string>();

  const applySnapshot = useEffectEvent((snapshot: DashboardSnapshot) => {
    if (!mountedRef.current) return;
    hasDataRef.current = true;
    setData(snapshot);
    setLoading(false);
    setError(undefined);
    setStreamState("live");
  });

  const reconnectStream = useEffectEvent(() => {
    streamRef.current?.close();
    streamRef.current = null;
    setStreamState("connecting");
    if (!hasDataRef.current) {
      setLoading(true);
    }
    setError(undefined);
    setStreamVersion((value) => value + 1);
  });

  useEffect(() => {
    mountedRef.current = true;
    if (!hasDataRef.current) {
      setLoading(true);
    }

    const source = new EventSource("/api/dashboard/stream");
    streamRef.current = source;

    source.onopen = () => {
      if (!mountedRef.current) return;
      setStreamState("live");
    };

    source.addEventListener("snapshot", (event) => {
      const message = event as MessageEvent<string>;
      try {
        applySnapshot(JSON.parse(message.data) as DashboardSnapshot);
      } catch {
        if (!mountedRef.current) return;
        setError("首页实时数据解析失败");
        setLoading(false);
        setStreamState("retrying");
      }
    });

    source.addEventListener("warning", (event) => {
      const message = event as MessageEvent<string>;
      try {
        const payload = JSON.parse(message.data) as { error?: string };
        if (!mountedRef.current) return;
        setError(payload.error || "首页实时推送异常");
        setLoading(false);
        setStreamState("retrying");
      } catch {
        if (!mountedRef.current) return;
        setError("首页实时推送异常");
        setLoading(false);
        setStreamState("retrying");
      }
    });

    source.onerror = () => {
      if (!mountedRef.current) return;
      setStreamState("retrying");
      if (!hasDataRef.current) {
        setError("首页实时连接断开，正在重连");
        setLoading(false);
      }
    };

    return () => {
      mountedRef.current = false;
      source.close();
      if (streamRef.current === source) {
        streamRef.current = null;
      }
    };
  }, [applySnapshot, streamVersion]);

  return {
    data,
    loading,
    streamState,
    error,
    reconnectStream,
  };
}
