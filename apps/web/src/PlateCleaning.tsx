import { useEffect, useRef, useState } from 'react'

type PlateCleaningProps = {
  actionId: string
  availableAt: string
  busy: boolean
  onComplete: (actionId: string) => void
}

const canvasWidth = 420
const canvasHeight = 190
const completionThreshold = 68

export function PlateCleaning({ actionId, availableAt, busy, onComplete }: PlateCleaningProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const drawingRef = useRef(false)
  const completedRef = useRef(false)
  const completionTimerRef = useRef<number | null>(null)
  const [progress, setProgress] = useState(0)

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
    setProgress(0)
    context.scale(ratio, ratio)
    context.save()
    context.beginPath()
    context.ellipse(canvasWidth / 2, canvasHeight / 2, 185, 78, 0, 0, Math.PI * 2)
    context.clip()
    context.fillStyle = 'rgba(91, 70, 52, .76)'
    context.fillRect(0, 0, canvasWidth, canvasHeight)

    const stains = [
      [76, 65, 28], [132, 112, 38], [205, 70, 33], [276, 118, 42], [344, 63, 30],
      [52, 142, 22], [181, 145, 25], [326, 147, 21], [250, 38, 17],
    ]
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
    context.fillStyle = 'rgba(236, 208, 145, .82)'
    context.font = '800 15px system-ui'
    context.textAlign = 'center'
    context.fillText('按住抹布，擦掉残渣', canvasWidth / 2, canvasHeight / 2 + 5)
    context.restore()
    context.globalCompositeOperation = 'destination-out'

    return () => {
      if (completionTimerRef.current !== null) window.clearTimeout(completionTimerRef.current)
    }
  }, [actionId])

  function wipe(event: React.PointerEvent<HTMLCanvasElement>) {
    if (!drawingRef.current || completedRef.current || busy) return
    const canvas = canvasRef.current
    const context = canvas?.getContext('2d', { willReadFrequently: true })
    if (!canvas || !context) return
    const bounds = canvas.getBoundingClientRect()
    const x = (event.clientX - bounds.left) * (canvasWidth / bounds.width)
    const y = (event.clientY - bounds.top) * (canvasHeight / bounds.height)
    context.beginPath()
    context.arc(x, y, 25, 0, Math.PI * 2)
    context.fill()
    if (event.timeStamp % 3 < 1) measure(context, canvas)
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
    <div className="plate-cleaning">
      <div className="plate-surface">
        <div className="clean-plate" aria-hidden="true"><span>✦</span></div>
        <canvas
          ref={canvasRef}
          className="plate-dirt"
          aria-label="脏盘子清洁区域"
          onPointerDown={(event) => {
            drawingRef.current = true
            event.currentTarget.setPointerCapture(event.pointerId)
            wipe(event)
          }}
          onPointerMove={wipe}
          onPointerUp={() => { drawingRef.current = false }}
          onPointerCancel={() => { drawingRef.current = false }}
        />
      </div>
      <div className="plate-progress"><span style={{ width: `${progress}%` }} /></div>
      <small>{progress >= completionThreshold ? '清洁完成，正在结算…' : `清洁度 ${progress}% · 达到 ${completionThreshold}% 完成`}</small>
    </div>
  )
}
