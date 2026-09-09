import { Button } from '../../../shared/ui/button'
import type { ReservationPage } from '../types'

type ReservationsPaginationProps = {
  page: Pick<ReservationPage, 'pagina' | 'paginas' | 'total'>
  isLoading: boolean
  onPageChange: (page: number) => void
}

export default function ReservationsPagination({
  page,
  isLoading,
  onPageChange,
}: ReservationsPaginationProps) {
  if (page.paginas <= 1) {
    return null
  }

  return (
    <nav
      className="flex flex-wrap items-center justify-between gap-3"
      aria-label="Paginación de reservas"
    >
      <p className="text-sm text-muted-foreground">
        Página {page.pagina} de {page.paginas} · {page.total} reservas
      </p>
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          disabled={page.pagina <= 1 || isLoading}
          onClick={() => onPageChange(Math.max(1, page.pagina - 1))}
        >
          Anterior
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={page.pagina >= page.paginas || isLoading}
          onClick={() => onPageChange(Math.min(page.paginas, page.pagina + 1))}
        >
          Siguiente
        </Button>
      </div>
    </nav>
  )
}
