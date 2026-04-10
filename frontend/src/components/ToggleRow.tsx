import { useTranslation } from 'react-i18next';

interface Props {
  label: string;
  tooltip: string;
  value: boolean;
  onChange: (value: boolean) => void;
}

export function ToggleRow({ label, tooltip, value, onChange }: Props) {
  const { t } = useTranslation();

  return (
    <div className="form-row" title={tooltip}>
      <label className="form-label">{label}</label>
      <div className="toggle-controls">
        <button
          className={`toggle-btn ${value ? 'active' : ''}`}
          onClick={() => onChange(true)}
          type="button"
        >
          {t('toggle.yes')}
        </button>
        <button
          className={`toggle-btn ${!value ? 'active' : ''}`}
          onClick={() => onChange(false)}
          type="button"
        >
          {t('toggle.no')}
        </button>
      </div>
    </div>
  );
}
