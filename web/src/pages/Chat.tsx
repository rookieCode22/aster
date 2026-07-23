import { useEffect, useState, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useWebSocket } from '../hooks/useWebSocket';
import { sessions } from '../api/client';

export default function Chat() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const { messages, connected, send, clearMessages } = useWebSocket(sessionId || null);
  const [input, setInput] = useState('');
  const [sessionTitle, setSessionTitle] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (sessionId) {
      sessions.get(sessionId).then((s) => setSessionTitle(s.title)).catch(() => {});
    }
  }, [sessionId]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = () => {
    if (!input.trim()) return;
    send(input.trim());
    setInput('');
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  if (!sessionId) {
    return (
      <div className="text-center py-20 text-gray-500">
        <p>No session selected.</p>
        <button onClick={() => navigate('/sessions')} className="mt-4 text-emerald-400 hover:underline">
          Go to Sessions
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between pb-4 border-b border-gray-800">
        <div>
          <button
            onClick={() => navigate('/sessions')}
            className="text-gray-400 hover:text-gray-200 text-sm transition mr-3"
          >
            ← Back
          </button>
          <h1 className="inline text-xl font-bold text-white">{sessionTitle || 'Chat'}</h1>
        </div>
        <div className="flex items-center gap-3">
          <span className={`w-2 h-2 rounded-full ${connected ? 'bg-emerald-400' : 'bg-red-400'}`} />
          <span className="text-xs text-gray-500">{connected ? 'Connected' : 'Disconnected'}</span>
        </div>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto py-4 space-y-4">
        {messages.length === 0 && (
          <div className="text-center py-20 text-gray-500">
            <p className="text-4xl mb-3">💬</p>
            <p>Send a message to start the security analysis.</p>
          </div>
        )}

        {messages.map((msg, i) => (
          <div
            key={msg.id || i}
            className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
          >
            <div
              className={`max-w-[80%] rounded-xl px-4 py-3 ${
                msg.role === 'user'
                  ? 'bg-emerald-600 text-white'
                  : 'bg-gray-800 text-gray-100 border border-gray-700'
              }`}
            >
              <div
                className="message-body text-sm whitespace-pre-wrap break-words"
                dangerouslySetInnerHTML={{
                  __html: msg.content
                    .replace(/```(\w*)\n([\s\S]*?)```/g, '<pre><code>$2</code></pre>')
                    .replace(/`([^`]+)`/g, '<code>$1</code>')
                    .replace(/\n/g, '<br/>'),
                }}
              />
              <p className="text-[10px] mt-2 opacity-50">
                {msg.timestamp ? new Date(msg.timestamp).toLocaleTimeString() : ''}
              </p>
            </div>
          </div>
        ))}

        {/* Typing indicator */}
        {messages.length > 0 && messages[messages.length - 1]?.role === 'user' && connected && (
          <div className="flex justify-start">
            <div className="bg-gray-800 border border-gray-700 rounded-xl px-4 py-3">
              <div className="flex gap-1">
                <span className="w-2 h-2 bg-gray-500 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                <span className="w-2 h-2 bg-gray-500 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                <span className="w-2 h-2 bg-gray-500 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
              </div>
            </div>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <div className="pt-4 border-t border-gray-800">
        <div className="flex gap-3">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ask Aster to analyze, scan, or test..."
            rows={2}
            className="flex-1 px-4 py-2.5 bg-gray-900/50 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:outline-none focus:border-emerald-500 transition resize-none"
            disabled={!connected}
          />
          <button
            onClick={handleSend}
            disabled={!connected || !input.trim()}
            className="px-5 py-2.5 bg-emerald-600 hover:bg-emerald-500 disabled:bg-gray-800 disabled:text-gray-600 text-white font-medium rounded-lg transition self-end"
          >
            Send
          </button>
        </div>
      </div>
    </div>
  );
}
