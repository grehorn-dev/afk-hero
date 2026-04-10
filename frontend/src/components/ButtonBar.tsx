import { useTranslation } from 'react-i18next';

interface Props {
  onOK: () => void;
  onCancel: () => void;
  onApply: () => void;
  onReset: () => void;
  applyDisabled: boolean;
}

export function ButtonBar({ onOK, onCancel, onApply, onReset, applyDisabled }: Props) {
  const { t } = useTranslation();

  return (
    <div className="button-bar">
      <button className="btn btn-primary" onClick={onOK} type="button">
        {t('button.ok')}
      </button>
      <button className="btn" onClick={onCancel} type="button">
        {t('button.cancel')}
      </button>
      <button className="btn btn-primary" onClick={onApply} disabled={applyDisabled} type="button">
        {t('button.apply')}
      </button>
      <button className="btn btn-danger" onClick={onReset} type="button">
        {t('button.reset')}
      </button>
    </div>
  );
}
