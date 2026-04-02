import { useEffect, useState } from "react";
import { createCopilotSession, sendCopilotMessage } from "../api";

export type ChatItem = {
  id: string;
  role: "user" | "assistant";
  content: string;
};

const INITIAL_MESSAGE: ChatItem = {
  id: crypto.randomUUID(),
  role: "assistant",
  content: "我可以帮你推荐应用、解释参数、读取项目日志和主机资源。",
};

export function useCopilotChat() {
  const [sessionId, setSessionId] = useState<string>();
  const [messages, setMessages] = useState<ChatItem[]>([INITIAL_MESSAGE]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [sessionError, setSessionError] = useState<string>();

  useEffect(() => {
    const init = async () => {
      try {
        const response = await createCopilotSession();
        setSessionId(response.id);
      } catch (error) {
        setSessionError(error instanceof Error ? error.message : "创建会话失败");
      }
    };

    void init();
  }, []);

  const send = async () => {
    const content = input.trim();
    if (!content || !sessionId) return;

    setSending(true);
    setInput("");

    const assistantId = crypto.randomUUID();
    setMessages((prev) => [
      ...prev,
      { id: crypto.randomUUID(), role: "user", content },
      { id: assistantId, role: "assistant", content: "" },
    ]);

    try {
      const response = await sendCopilotMessage(sessionId, content);
      if (!response.ok || !response.body) {
        throw new Error("Copilot 请求失败");
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const chunks = buffer.split("\n\n");
        buffer = chunks.pop() || "";

        for (const chunk of chunks) {
          const lines = chunk.split("\n");
          const event = lines.find((line) => line.startsWith("event:"))?.replace("event:", "").trim();
          const dataLine = lines.find((line) => line.startsWith("data:"))?.replace("data:", "").trim();
          if (!event || !dataLine) continue;

          const payload = JSON.parse(dataLine) as Record<string, string>;
          if (event === "chunk") {
            setMessages((prev) =>
              prev.map((item) =>
                item.id === assistantId
                  ? { ...item, content: item.content + (payload.content || "") }
                  : item,
              ),
            );
          }
          if (event === "error") {
            throw new Error(payload.error || "Copilot 出错");
          }
        }
      }
    } catch (error) {
      setMessages((prev) =>
        prev.map((item) =>
          item.id === assistantId
            ? { ...item, content: error instanceof Error ? error.message : "Copilot 调用失败" }
            : item,
        ),
      );
    } finally {
      setSending(false);
    }
  };

  return {
    sessionId,
    messages,
    input,
    setInput,
    sending,
    sessionError,
    send,
  };
}
