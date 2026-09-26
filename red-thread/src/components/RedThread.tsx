'use client';

import { useEffect, useRef } from 'react';

/**
 * The red thread of fate (月老的红线). It is drawn through every `[data-knot]`
 * inside its parent and unspools as the reader scrolls down; knots light up
 * once the thread reaches them. The dotted guide shows the road still ahead.
 */
export function RedThread() {
  const svgRef = useRef<SVGSVGElement>(null);

  useEffect(() => {
    const svg = svgRef.current;
    const host = svg?.parentElement;
    if (!svg || !host) return;
    const trail = svg.querySelector<SVGPathElement>('.thread-trail')!;
    const guide = svg.querySelector<SVGPathElement>('.thread-guide')!;
    let length = 0;
    let knots: { el: Element; y: number }[] = [];
    let frame = 0;

    const update = () => {
      frame = 0;
      const rect = host.getBoundingClientRect();
      const reach = window.innerHeight * 0.72 - rect.top;
      const progress = Math.min(Math.max(reach / rect.height, 0), 1);
      trail.style.strokeDashoffset = String(length * (1 - progress));
      for (const knot of knots) knot.el.classList.toggle('lit', knot.y <= reach);
    };

    const build = () => {
      const hostRect = host.getBoundingClientRect();
      const width = host.clientWidth;
      const height = host.offsetHeight;
      svg.setAttribute('viewBox', `0 0 ${width} ${height}`);
      const points = Array.from(host.querySelectorAll('[data-knot]')).map((el) => {
        const r = el.getBoundingClientRect();
        return { el, x: r.left + r.width / 2 - hostRect.left, y: r.top + r.height / 2 - hostRect.top };
      });
      if (points.length === 0) return;
      const amplitude = width < 760 ? 10 : 42;
      let d = `M ${points[0].x} 0`;
      let prev = { x: points[0].x, y: 0 };
      [...points, { x: points[points.length - 1].x, y: height }].forEach((point, i) => {
        const dy = point.y - prev.y;
        const side = i % 2 === 0 ? 1 : -1;
        d += ` C ${prev.x + amplitude * side} ${prev.y + dy * 0.35}, ${point.x - amplitude * side} ${prev.y + dy * 0.65}, ${point.x} ${point.y}`;
        prev = point;
      });
      trail.setAttribute('d', d);
      guide.setAttribute('d', d);
      length = trail.getTotalLength();
      trail.style.strokeDasharray = `${length} ${length}`;
      knots = points.map((p) => ({ el: p.el, y: p.y }));
      update();
    };

    const schedule = () => {
      if (!frame) frame = requestAnimationFrame(update);
    };
    const resize = new ResizeObserver(() => build());
    resize.observe(host);
    window.addEventListener('scroll', schedule, { passive: true });
    window.addEventListener('resize', schedule);
    document.fonts?.ready.then(build).catch(() => {});
    build();
    return () => {
      resize.disconnect();
      window.removeEventListener('scroll', schedule);
      window.removeEventListener('resize', schedule);
      cancelAnimationFrame(frame);
    };
  }, []);

  return (
    <svg ref={svgRef} className="thread" aria-hidden preserveAspectRatio="none">
      <path className="thread-guide" />
      <path className="thread-trail" />
    </svg>
  );
}
