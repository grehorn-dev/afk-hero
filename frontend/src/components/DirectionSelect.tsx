import { useTranslation } from 'react-i18next';
import { DIRECTIONS } from '../types/settings';
import type { Direction } from '../types/settings';

interface Props {
  value: Direction;
  onChange: (direction: Direction) => void;
}

export function DirectionSelect({ value, onChange }: Props) {
  const { t } = useTranslation();

  return (
    <div className="form-row" title={t('tooltip.direction')}>
      <label className="form-label">{t('settings.direction')}</label>
      <select
        className="form-select"
        value={value}
        onChange={(e) => onChange(e.target.value as Direction)}
      >
        {DIRECTIONS.map((dir) => (
          <option key={dir} value={dir}>
            {t(`direction.${dir}`)}
          </option>
        ))}
      </select>
    </div>
  );
}
