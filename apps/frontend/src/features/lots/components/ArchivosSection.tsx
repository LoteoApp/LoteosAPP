import { useEffect, useRef, useState, type ChangeEvent } from 'react'
import { Upload, X } from 'lucide-react'
import { Button } from '../../../shared/ui/button'
import { fetchArchivoContent } from '../api/archivos'
import { useArchivos, type ArchivoTarget } from '../hooks/use-archivos'
import type { Archivo, ArchivoCategoria } from '../types'

const ARCHIVO_ACCEPT = 'image/jpeg,image/png,image/webp,application/pdf'

// The categoria the backend stores is an implementation detail — the person
// uploading never chooses "foto" vs. "plano" (that word already means the
// DXF-derived plan elsewhere in this screen), so it's inferred from the
// file's own type instead of asked for.
function categoriaFor(file: File): ArchivoCategoria {
  return file.type.startsWith('image/') ? 'foto' : 'plano'
}

type ArchivosSectionProps = {
  target: ArchivoTarget
  accessToken: string | null
  canEdit: boolean
}

export default function ArchivosSection({ target, accessToken, canEdit }: ArchivosSectionProps) {
  const { archivos, isLoading, isSubmitting, error, upload, remove } = useArchivos(target, accessToken)
  const fileInputRef = useRef<HTMLInputElement>(null)

  async function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    // Let the same file be picked again right after a failed upload.
    event.target.value = ''
    if (file) {
      await upload(categoriaFor(file), file)
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs font-medium text-muted-foreground">Fotos y documentos</p>

      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}

      {canEdit && (
        <div>
          <Button
            type="button"
            variant="outline"
            disabled={isSubmitting}
            onClick={() => fileInputRef.current?.click()}
          >
            <Upload aria-hidden />
            Subir archivo
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept={ARCHIVO_ACCEPT}
            onChange={handleFileChange}
            disabled={isSubmitting}
            aria-label="Subir archivo"
            className="hidden"
          />
        </div>
      )}

      {isLoading && <p className="text-sm text-muted-foreground">Cargando archivos…</p>}
      {!isLoading && archivos.length === 0 && (
        <p className="text-sm text-muted-foreground">Todavía no hay archivos.</p>
      )}

      {archivos.length > 0 && (
        // Fixed-height strip: with more archivos it scrolls sideways instead
        // of pushing the rest of the panel down.
        <ul className="flex gap-2 overflow-x-auto pb-1">
          {archivos.map((archivo) => (
            <ArchivoThumbnail
              key={archivo.id}
              archivo={archivo}
              loteoId={target.loteoId}
              accessToken={accessToken}
              canEdit={canEdit}
              disabled={isSubmitting}
              onDelete={() => remove(archivo.id)}
            />
          ))}
        </ul>
      )}
    </div>
  )
}

type ArchivoThumbnailProps = {
  archivo: Archivo
  loteoId: string
  accessToken: string | null
  canEdit: boolean
  disabled: boolean
  onDelete: () => void
}

function ArchivoThumbnail({ archivo, loteoId, accessToken, canEdit, disabled, onDelete }: ArchivoThumbnailProps) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)

  useEffect(() => {
    if (!accessToken) {
      return
    }

    let cancelled = false
    let objectUrl: string | null = null

    fetchArchivoContent(loteoId, archivo.id, accessToken)
      .then((blob) => {
        if (cancelled) {
          return
        }
        objectUrl = URL.createObjectURL(blob)
        setPreviewUrl(objectUrl)
      })
      .catch(() => {
        // The thumbnail still shows its fallback badge without a preview.
      })

    return () => {
      cancelled = true
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl)
      }
    }
  }, [loteoId, archivo.id, accessToken])

  const isImage = archivo.mimeType.startsWith('image/')

  const content = isImage && previewUrl ? (
    <img src={previewUrl} alt={archivo.nombreOriginal} className="size-full object-cover" />
  ) : (
    <div className="flex size-full items-center justify-center bg-muted text-[0.65rem] font-medium text-muted-foreground">
      {isImage ? 'IMG' : 'PDF'}
    </div>
  )

  return (
    <li className="relative size-20 shrink-0">
      {previewUrl ? (
        <a
          href={previewUrl}
          target="_blank"
          rel="noreferrer"
          title={archivo.nombreOriginal}
          className="block size-full overflow-hidden rounded-md border border-border"
        >
          {content}
        </a>
      ) : (
        <div title={archivo.nombreOriginal} className="size-full overflow-hidden rounded-md border border-border">
          {content}
        </div>
      )}

      {canEdit && (
        <Button
          type="button"
          variant="outline"
          size="icon-xs"
          disabled={disabled}
          onClick={onDelete}
          aria-label={`Eliminar ${archivo.nombreOriginal}`}
          className="absolute -top-1.5 -right-1.5 rounded-full bg-background"
        >
          <X aria-hidden />
        </Button>
      )}
    </li>
  )
}
