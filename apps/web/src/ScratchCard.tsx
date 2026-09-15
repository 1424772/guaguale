import { useEffect, useRef, useState } from 'react'

type ScratchCardProps = {
  symbols: string[]
  onComplete: () => void
}

const symbolVisuals: Record<string, { emoji: string; label: string }> = {
  '狗头金币': { emoji: '🐶', label: '狗头金币' },
  '钞票': { emoji: '💵', label: '钞票' },
  '碎钻石': { emoji: '💎', label: '碎钻石' },
  '钞票堆': { emoji: '💰', label: '钞票堆' },
}

export function ScratchCard({ symbols, onComplete }: ScratchCardProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const drawingRef = useRef(false)
  const completedRef = useRef(false)
  const [progress, setProgress] = useState(0)

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ratio = Math.min(window.devicePixelRatio || 1, 2)
    const width = 720
    const height = 220
    canvas.width = width * ratio
    canvas.height = height * ratio
    const context = canvas.getContext('2d', { willReadFrequently: true })
    if (!context) return
    context.scale(ratio, ratio)
    const gradient = context.createLinearGradient(0, 0, width, height)
    gradient.addColorStop(0, '#d9d7ce')
    gradient.addColorStop(0.45, '#a7a8a5')
    gradient.addColorStop(1, '#e7e2d4')
    context.fillStyle = gradient
    context.fillRect(0, 0, width, height)
    context.fillStyle = 'rgba(60, 65, 66, .38)'
    context.font = '700 26px system-ui'
    context.textAlign = 'center'
    context.fillText('按住并刮开', width / 2, height / 2 + 8)
    context.globalCompositeOperation = 'destination-out'
  }, [])

  function scratch(event: React.PointerEvent<HTMLCanvasElement>) {
    if (!drawingRef.current || completedRef.current) return
    const canvas = canvasRef.current
    const context = canvas?.getContext('2d', { willReadFrequently: true })
    if (!canvas || !context) return
    const bounds = canvas.getBoundingClientRect()
    const x = (event.clientX - bounds.left) * (720 / bounds.width)
    const y = (event.clientY - bounds.top) * (220 / bounds.height)
    context.beginPath()
    context.arc(x, y, 30, 0, Math.PI * 2)
    context.fill()
    if (event.timeStamp % 4 < 1) measureProgress(context, canvas)
  }

  function measureProgress(context: CanvasRenderingContext2D, canvas: HTMLCanvasElement) {
    const pixels = context.getImageData(0, 0, canvas.width, canvas.height).data
    let transparent = 0
    let sampled = 0
    for (let index = 3; index < pixels.length; index += 4 * 24) {
      sampled++
      if (pixels[index] < 64) transparent++
    }
    const nextProgress = Math.round((transparent / sampled) * 100)
    setProgress(nextProgress)
    if (nextProgress >= 45) finish(context, canvas)
  }

  function finish(context: CanvasRenderingContext2D, canvas: HTMLCanvasElement) {
    if (completedRef.current) return
    completedRef.current = true
    context.clearRect(0, 0, canvas.width, canvas.height)
    setProgress(100)
    onComplete()
  }

  return (
    <div className="scratch-card">
      <div className="ticket-heading">
        <span>零钱小票</span>
        <strong>三格中两格相同即中奖</strong>
      </div>
      <div className="scratch-stage">
        <div className="symbols" aria-hidden={!completedRef.current}>
          {symbols.map((symbol, index) => {
            const visual = symbolVisuals[symbol] ?? { emoji: '✦', label: symbol }
            return (
              <div className="symbol" key={`${symbol}-${index}`}>
                <span>{visual.emoji}</span>
                <small>{visual.label}</small>
              </div>
            )
          })}
        </div>
        <canvas
          ref={canvasRef}
          className="scratch-layer"
          onPointerDown={(event) => {
            drawingRef.current = true
            event.currentTarget.setPointerCapture(event.pointerId)
            scratch(event)
          }}
          onPointerMove={scratch}
          onPointerUp={() => { drawingRef.current = false }}
          onPointerCancel={() => { drawingRef.current = false }}
          aria-label="刮奖区域"
        />
      </div>
      <div className="scratch-progress" aria-live="polite">
        <span style={{ width: `${progress}%` }} />
      </div>
      <p>刮开45%后自动揭晓完整结果</p>
    </div>
  )
}
