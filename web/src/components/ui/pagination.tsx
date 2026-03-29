interface PaginationProps {
  currentPage: number;
  totalPages: number;
  totalItems: number;
  onPageChange: (page: number) => void;
}

function getPageNumbers(current: number, total: number): (number | "...")[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1);
  }
  const pages: (number | "...")[] = [1];
  if (current > 3) pages.push("...");
  const start = Math.max(2, current - 1);
  const end = Math.min(total - 1, current + 1);
  for (let i = start; i <= end; i++) pages.push(i);
  if (current < total - 2) pages.push("...");
  if (total > 1) pages.push(total);
  return pages;
}

const btnBase =
  "inline-flex items-center justify-center rounded-[var(--radius)] text-[13px] font-medium transition-colors disabled:opacity-40 disabled:pointer-events-none";

export function Pagination({
  currentPage,
  totalPages,
  totalItems,
  onPageChange,
}: PaginationProps) {
  if (totalPages <= 1) return null;

  const pages = getPageNumbers(currentPage, totalPages);

  return (
    <div className="flex items-center justify-between pt-2">
      {/* Item count */}
      <span className="text-[13px] text-[var(--color-text-muted)]">
        {totalItems} items
      </span>

      {/* Page controls */}
      <div className="flex items-center gap-1.5">
        {/* Prev */}
        <button
          className={`${btnBase} w-8 h-8 bg-[var(--color-bg-accent)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]`}
          disabled={currentPage === 1}
          onClick={() => onPageChange(currentPage - 1)}
          aria-label="Previous page"
        >
          ←
        </button>

        {/* Page numbers */}
        {pages.map((p, i) =>
          p === "..." ? (
            <span
              key={`ellipsis-${i}`}
              className="w-8 h-8 inline-flex items-center justify-center text-[13px] text-[var(--color-text-muted)]"
            >
              ...
            </span>
          ) : (
            <button
              key={p}
              className={`${btnBase} w-8 h-8 ${
                p === currentPage
                  ? "bg-[var(--color-primary)] text-[var(--color-bg)]"
                  : "bg-[var(--color-bg-accent)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
              onClick={() => onPageChange(p)}
              aria-label={`Page ${p}`}
              aria-current={p === currentPage ? "page" : undefined}
            >
              {p}
            </button>
          ),
        )}

        {/* Next */}
        <button
          className={`${btnBase} w-8 h-8 bg-[var(--color-bg-accent)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]`}
          disabled={currentPage === totalPages}
          onClick={() => onPageChange(currentPage + 1)}
          aria-label="Next page"
        >
          →
        </button>
      </div>
    </div>
  );
}
