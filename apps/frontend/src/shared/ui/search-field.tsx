import { Input } from './input'
import { Label } from './label'

type SearchFieldProps = {
  id: string
  label?: string
  placeholder?: string
  value: string
  onChange: (value: string) => void
  className?: string
  inputClassName?: string
}

export function SearchField({
  id,
  label = 'Buscar',
  placeholder,
  value,
  onChange,
  className,
  inputClassName,
}: SearchFieldProps) {
  return (
    <div className={className ?? 'flex flex-col gap-1.5'}>
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="search"
        placeholder={placeholder}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className={inputClassName}
      />
    </div>
  )
}
