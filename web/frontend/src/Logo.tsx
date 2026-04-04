export default function Logo({ size = 28 }: { size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      {/* Two rounded squares overlapping — representing two systems being glued */}
      <rect x="2" y="6" width="14" height="14" rx="3" fill="#4f6ef7" opacity="0.85" />
      <rect x="16" y="12" width="14" height="14" rx="3" fill="#4f6ef7" opacity="0.55" />
      {/* Connecting bridge — the "glue" */}
      <path
        d="M14 13 L18 13 L18 19 L14 19"
        fill="#4f6ef7"
      />
      {/* Arrow showing data flow */}
      <path
        d="M8 13 L24 13"
        stroke="#fff"
        strokeWidth="1.5"
        strokeLinecap="round"
      />
      <path
        d="M21 10.5 L24 13 L21 15.5"
        stroke="#fff"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
    </svg>
  );
}
