import { useState } from "react";
import { Copy, Check } from "lucide-react";

function copyText(text: string): Promise<void> {
  // navigator.clipboard requires HTTPS or localhost on some browsers.
  // Fallback to execCommand for HTTP environments.
  if (navigator.clipboard?.writeText) {
    return navigator.clipboard.writeText(text).catch(() => execCommandCopy(text));
  }
  return execCommandCopy(text);
}

function execCommandCopy(text: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.style.position = "fixed";
    textarea.style.left = "-9999px";
    document.body.appendChild(textarea);
    textarea.select();
    try {
      document.execCommand("copy");
      resolve();
    } catch {
      reject(new Error("Copy failed"));
    } finally {
      document.body.removeChild(textarea);
    }
  });
}

export function CopyButton({ text, title }: { text: string; title?: string }) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    copyText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  }

  return (
    <button
      onClick={handleCopy}
      title={copied ? "Copied!" : (title ?? "Copy")}
      className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-mono transition-colors hover:bg-[var(--color-accent)]"
      style={{ color: "var(--color-muted-foreground)" }}
    >
      {copied ? (
        <Check className="h-3 w-3 text-green-500" />
      ) : (
        <Copy className="h-3 w-3" />
      )}
      <span className="truncate max-w-[160px] inline-block align-middle">
        {text.replace(window.location.origin, "")}
      </span>
    </button>
  );
}
