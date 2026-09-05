import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ArchivosSection from './ArchivosSection'
import * as archivosApi from '../api/archivos'
import type { Archivo } from '../types'

const archivo: Archivo = {
  id: 'archivo-1',
  categoria: 'foto',
  nombreOriginal: 'foto.jpg',
  mimeType: 'image/jpeg',
  hashSha256: 'abc123',
  fechaCreacion: '2026-01-01T00:00:00Z',
}

const loteoTarget = { kind: 'loteo' as const, loteoId: 'loteo-1' }

beforeEach(() => {
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:preview')
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('ArchivosSection', () => {
  it('shows an empty state when there are no archivos', async () => {
    vi.spyOn(archivosApi, 'listLoteoArchivos').mockResolvedValue([])

    render(<ArchivosSection target={loteoTarget} accessToken="tok" canEdit />)

    expect(await screen.findByText('Todavía no hay archivos.')).toBeInTheDocument()
  })

  it('shows an archivo as a thumbnail linking to its preview once fetched', async () => {
    vi.spyOn(archivosApi, 'listLoteoArchivos').mockResolvedValue([archivo])
    vi.spyOn(archivosApi, 'fetchArchivoContent').mockResolvedValue(new Blob(['x'], { type: 'image/jpeg' }))

    render(<ArchivosSection target={loteoTarget} accessToken="tok" canEdit />)

    await waitFor(() => expect(screen.getByRole('img', { name: 'foto.jpg' })).toHaveAttribute('src', 'blob:preview'))
    expect(screen.getByRole('link', { name: 'foto.jpg' })).toHaveAttribute('href', 'blob:preview')
  })

  it('shows a plano without an image preview as a generic file badge', async () => {
    vi.spyOn(archivosApi, 'listLoteoArchivos').mockResolvedValue([
      { ...archivo, categoria: 'plano', mimeType: 'application/pdf', nombreOriginal: 'plano.pdf' },
    ])
    vi.spyOn(archivosApi, 'fetchArchivoContent').mockRejectedValue(new Error('boom'))

    render(<ArchivosSection target={loteoTarget} accessToken="tok" canEdit />)

    expect(await screen.findByText('PDF')).toBeInTheDocument()
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
    // No preview came back, so the badge isn't wrapped in a link.
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })

  it('uploads a selected file with the chosen categoria', async () => {
    const user = userEvent.setup()
    vi.spyOn(archivosApi, 'listLoteoArchivos').mockResolvedValue([])
    const upload = vi.spyOn(archivosApi, 'uploadLoteoArchivo').mockResolvedValue(archivo)

    render(<ArchivosSection target={loteoTarget} accessToken="tok" canEdit />)
    await screen.findByText('Todavía no hay archivos.')

    await user.click(screen.getByRole('button', { name: 'Plano' }))
    const file = new File(['bytes'], 'plano.pdf', { type: 'application/pdf' })
    await user.upload(screen.getByLabelText('Subir archivo'), file)

    await waitFor(() => expect(upload).toHaveBeenCalledWith('loteo-1', 'plano', file, 'tok'))
  })

  it('removes an archivo when clicking its delete button', async () => {
    const user = userEvent.setup()
    vi.spyOn(archivosApi, 'listLoteoArchivos').mockResolvedValue([archivo])
    vi.spyOn(archivosApi, 'fetchArchivoContent').mockResolvedValue(new Blob(['x'], { type: 'image/jpeg' }))
    const remove = vi.spyOn(archivosApi, 'deleteArchivo').mockResolvedValue(undefined)

    render(<ArchivosSection target={loteoTarget} accessToken="tok" canEdit />)
    await screen.findByRole('img', { name: 'foto.jpg' })

    await user.click(screen.getByRole('button', { name: 'Eliminar foto.jpg' }))

    await waitFor(() => expect(remove).toHaveBeenCalledWith('loteo-1', 'archivo-1', 'tok'))
    await waitFor(() => expect(screen.queryByRole('img', { name: 'foto.jpg' })).not.toBeInTheDocument())
  })

  it('shows the error message when loading fails', async () => {
    vi.spyOn(archivosApi, 'listLoteoArchivos').mockRejectedValue(new Error('boom'))

    render(<ArchivosSection target={loteoTarget} accessToken="tok" canEdit />)

    expect(await screen.findByRole('alert')).toBeInTheDocument()
  })

  it('hides the upload control and delete buttons in read-only mode', async () => {
    vi.spyOn(archivosApi, 'listLoteoArchivos').mockResolvedValue([archivo])
    vi.spyOn(archivosApi, 'fetchArchivoContent').mockResolvedValue(new Blob(['x'], { type: 'image/jpeg' }))

    render(<ArchivosSection target={loteoTarget} accessToken="tok" canEdit={false} />)

    await screen.findByRole('img', { name: 'foto.jpg' })
    expect(screen.queryByLabelText('Subir archivo')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Eliminar foto.jpg' })).not.toBeInTheDocument()
  })
})
