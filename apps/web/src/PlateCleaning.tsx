import { useEffect, useRef, useState } from 'react'

type PlateCleaningProps = {
  actionId: string
  sequence: number
  plateLimit: number
  availableAt: string
  busy: boolean
  onComplete: (actionId: string) => void
}

const canvasWidth = 420
const canvasHeight = 190
const completionThreshold = 68

export function PlateCleaning({ actionId, sequence, plateLimit, availableAt, busy, onComplete }: PlateCleaningProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const drawingRef = useRef(false)
  const completedRef = useRef(false)
  const movementCountRef = useRef(0)
  const completionTimerRef = useRef<number | null>(null)
  const [progress, setProgress] = useState(0)
  const [sponge, setSponge] = useState({ x: canvasWidth / 2, y: canvasHeight / 2, visible: false, active: false })

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ratio = Math.min(window.devicePixelRatio || 1, 2)
    canvas.width = canvasWidth * ratio
    canvas.height = canvasHeight * ratio
    const context = canvas.getContext('2d', { willReadFrequently: true })
    if (!context) return

    completedRef.current = false
    drawingRef.current = false
    movementCountRef.current = 0
    setProgress(0)
    context.scale(ratio, ratio)
    context.save()
    context.beginPath()
    context.ellipse(canvasWidth / 2, canvasHeight / 2, 185, 78, 0, 0, Math.PI * 2)
    context.clip()
    context.fillStyle = 'rgba(91, 70, 52, .76)'
    context.fillRect(0, 0, canvasWidth, canvasHeight)

    const seed = [...actionId].reduce((value, character) => value + character.charCodeAt(0), 0)
    const stains = Array.from({ length: 11 }, (_, index) => {
      const angle = ((index * 137 + seed) % 360) * Math.PI / 180
      const distance = 22 + ((index * 31 + seed) % 120)
      return [
        canvasWidth / 2 + Math.cos(angle) * distance,
        canvasHeight / 2 + Math.sin(angle) * distance * .42,
        15 + ((index * 17 + seed) % 24),
      ]
    })
    for (const [x, y, radius] of stains) {
      const stain = context.createRadialGradient(x, y, 3, x, y, radius)
      stain.addColorStop(0, 'rgba(62, 35, 19, .98)')
      stain.addColorStop(.58, 'rgba(109, 67, 34, .94)')
      stain.addColorStop(1, 'rgba(78, 49, 29, .1)')
      context.fillStyle = stain
      context.beginPath()
      context.arc(x, y, radius, 0, Math.PI * 2)
      context.fill()
    }
    context.strokeStyle = 'rgba(63, 37, 20, .66)'
    context.lineWidth = 10
    context.lineCap = 'round'
    context.beginPath()
    context.moveTo(110, 80)
    context.bezierCurveTo(165, 45, 238, 138, 320, 83)
    context.stroke()
    context.restore()
    context.globalCompositeOperation = 'destination-out'

    return () => {
      if (completionTimerRef.current !== null) window.clearTimeout(completionTimerRef.current)
    }
  }, [actionId])

  function wipe(event: React.PointerEvent<HTMLCanvasElement>) {
    const canvas = canvasRef.current
    if (!canvas) return
    const bounds = canvas.getBoundingClientRect()
    const x = (event.clientX - bounds.left) * (canvasWidth / bounds.width)
    const y = (event.clientY - bounds.top) * (canvasHeight / bounds.height)
    setSponge({ x, y, visible: true, active: drawingRef.current })
    if (!drawingRef.current || completedRef.current || busy) return
    const context = canvas.getContext('2d', { willReadFrequently: true })
    if (!context) return
    context.beginPath()
    context.arc(x, y, 25, 0, Math.PI * 2)
    context.fill()
    movementCountRef.current++
    if (movementCountRef.current % 3 === 0) measure(context, canvas)
  }

  function measure(context: CanvasRenderingContext2D, canvas: HTMLCanvasElement) {
    const pixels = context.getImageData(0, 0, canvas.width, canvas.height).data
    const ratio = canvas.width / canvasWidth
    let transparent = 0
    let sampled = 0
    const step = Math.max(8, Math.round(10 * ratio))
    for (let y = 0; y < canvas.height; y += step) {
      for (let x = 0; x < canvas.width; x += step) {
        const logicalX = x / ratio
        const logicalY = y / ratio
        const insidePlate = ((logicalX - canvasWidth / 2) / 185) ** 2 + ((logicalY - canvasHeight / 2) / 78) ** 2 <= 1
        if (!insidePlate) continue
        sampled++
        const alphaIndex = (y * canvas.width + x) * 4 + 3
        if (pixels[alphaIndex] < 64) transparent++
      }
    }
    const next = Math.round((transparent / sampled) * 100)
    setProgress(next)
    if (next >= completionThreshold) finish(context, canvas)
  }

  function finish(context: CanvasRenderingContext2D, canvas: HTMLCanvasElement) {
    if (completedRef.current) return
    completedRef.current = true
    drawingRef.current = false
    context.clearRect(0, 0, canvas.width, canvas.height)
    setProgress(100)
    const delay = Math.max(0, new Date(availableAt).getTime() - Date.now()) + 250
    completionTimerRef.current = window.setTimeout(() => onComplete(actionId), delay)
  }

  return (
    <section className={`plate-cleaning ${progress >= completionThreshold ? 'plate-cleaned' : ''}`} aria-labelledby="wash-title">
      <header className="wash-header">
        <div>
          <span className="eyebrow">DISHWASHING MINI GAME</span>
          <h2 id="wash-title">把盘子擦干净</h2>
          <p>按住海绵来回擦洗残渣，清洁度达到 {completionThreshold}% 即可获得 5 金币。</p>
        </div>
        <div className="wash-count"><span>今日进度</span><strong>{sequence} / {plateLimit}</strong></div>
      </header>

      <div className="sink-station">
        <div className="faucet" aria-hidden="true"><i /><span /></div>
        <div className="water-ripples" aria-hidden="true"><i /><i /><i /></div>
        <div className="plate-surface">
          <div className="clean-plate" aria-hidden="true"><span>✦</span></div>
          <div className="foam-layer" aria-hidden="true" style={{ opacity: Math.min(progress / 80, .82) }}>
            {Array.from({ length: 14 }, (_, index) => <i key={index} />)}
          </div>
          <canvas
            ref={canvasRef}
            className="plate-dirt"
            aria-label="脏盘子清洁区域"
            onPointerEnter={wipe}
            onPointerLeave={() => setSponge((current) => ({ ...current, visible: false, active: false }))}
            onPointerDown={(event) => {
              drawingRef.current = true
              event.currentTarget.setPointerCapture(event.pointerId)
              wipe(event)
            }}
            onPointerMove={wipe}
            onPointerUp={() => {
              drawingRef.current = false
              setSponge((current) => ({ ...current, active: false }))
              const canvas = canvasRef.current
              const context = canvas?.getContext('2d', { willReadFrequently: true })
              if (canvas && context && !completedRef.current) measure(context, canvas)
            }}
            onPointerCancel={() => {
              drawingRef.current = false
              setSponge((current) => ({ ...current, active: false }))
            }}
          />
          {sponge.visible && (
            <div
              className={`sponge-tool ${sponge.active ? 'active' : ''}`}
              aria-hidden="true"
              style={{ left: `${(sponge.x / canvasWidth) * 100}%`, top: `${(sponge.y / canvasHeight) * 100}%` }}
            ><span /></div>
          )}
          <div className="wash-instruction" aria-hidden="true">按住并来回擦洗</div>
        </div>
      </div>

      <div className="wash-footer" aria-live="polite">
        <div className="plate-progress"><span style={{ width: `${progress}%` }} /></div>
        <strong>{progress}%</strong>
        <span>{progress >= completionThreshold ? '洗干净了，正在结算奖励…' : progress > 0 ? '继续擦，还有残渣' : '从盘面任意位置开始擦洗'}</span>
      </div>
    </section>
  )
}
