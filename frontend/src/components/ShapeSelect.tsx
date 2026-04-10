import { useTranslation } from 'react-i18next';
import { SHAPES } from '../types/settings';
import type { Shape } from '../types/settings';

interface Props {
  value: Shape;
  onChange: (shape: Shape) => void;
}

export function ShapeSelect({ value, onChange }: Props) {
  const { t } = useTranslation();

  return (
    <div className="form-row" title={t('tooltip.shape')}>
      <label className="form-label">{t('settings.shape')}</label>
      <select
        className="form-select"
        value={value}
        onChange={(e) => onChange(e.target.value as Shape)}
      >
        {SHAPES.map((shape) => (
          <option key={shape} value={shape}>
            {t(`shape.${shape}`)}
          </option>
        ))}
      </select>
    </div>
  );
}
