import { useTranslation } from 'react-i18next';

interface Props {
  progress: number;
  statusKey: string;
}

export function StatusBar({ progress, statusKey }: Props) {
  const { t } = useTranslation();
  const percent = Math.max(0, Math.min(progress, 1)) * 100;

  return (
    <div className="status-bar">
      <div className="progress-container">
        <div className="progress-bar" style={{ width: `${percent}%` }} />
      </div>
      <div className="status-text">{t(statusKey)}</div>
    </div>
  );
}
