import { SendOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Input, Space, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { apiRequest } from "../lib/api";
import type { CopilotSession } from "../lib/types";

type ChatItem = {
  id: string;
  role: "user" | "assistant";
  content: string;
};

export function CopilotPage() {
  const [session, setSession] = useState<CopilotSession | null>(null);
  const [messages, setMessages] = useState<ChatItem[]>([
    {
      id: crypto.randomUUID(),
      role: "assistant",
      content:
        "我可以帮你推荐应用、解释参数、读取项目日志和主机资源，并在需要时生成审批单。",
    },
  ]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [sessionError, setSessionError] = useState<string>();

  useEffect(() => {
    const init = async () => {
      try {
        const response = await apiRequest<CopilotSession>("/api/copilot/sessions", {
          method: "POST",
        });
        setSession(response);
      } catch (err) {
        setSessionError(err instanceof Error ? err.message : "创建会话失败");
      }
    };
    void init();
  }, []);

  const send = async () => {
    const content = input.trim();
    if (!content || !session) return;

    setSending(true);
    setInput("");

    const assistantID = crypto.randomUUID();
    setMessages((prev) => [
      ...prev,
      { id: crypto.randomUUID(), role: "user", content },
      { id: assistantID, role: "assistant", content: "" },
    ]);

    try {
      const response = await fetch(`/api/copilot/sessions/${session.id}/messages`, {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ message: content }),
      });

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
                item.id === assistantID
                  ? { ...item, content: item.content + (payload.content || "") }
                  : item,
              ),
            );
          }
          if (event === "action") {
            message.success(`AI 已生成审批单：${payload.summary}`);
          }
          if (event === "error") {
            message.error(payload.error || "Copilot 出错");
          }
        }
      }
    } catch (err) {
      setMessages((prev) =>
        prev.map((item) =>
          item.id === assistantID
            ? { ...item, content: err instanceof Error ? err.message : "Copilot 调用失败" }
            : item,
        ),
      );
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="page-grid">
      <div>
        <Typography.Title className="page-title">Camo Copilot</Typography.Title>
        <Typography.Paragraph className="page-subtitle">
          Copilot 只做读操作和提案，不直接执行写入。所有动作都会被转成审批单。
        </Typography.Paragraph>
      </div>

      {sessionError ? <Alert type="error" message={sessionError} /> : null}

      <Card className="glass-card">
        <div className="copilot-stream">
          {messages.map((item) => (
            <div key={item.id} className={`chat-bubble ${item.role}`}>
              {item.content || "正在思考..."}
            </div>
          ))}
        </div>
      </Card>

      <Card className="glass-card">
        <Space.Compact style={{ width: "100%" }}>
          <Input.TextArea
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder="例如：帮我部署一个 WordPress，或者这个项目为什么起不来？"
            autoSize={{ minRows: 2, maxRows: 5 }}
          />
          <Button type="primary" icon={<SendOutlined />} loading={sending} onClick={() => void send()}>
            发送
          </Button>
        </Space.Compact>
      </Card>
    </div>
  );
}
