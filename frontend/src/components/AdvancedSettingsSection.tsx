import { useTranslation } from 'react-i18next';
import type { Direction, NumericRange, Shape } from '../types/settings';
import { DirectionSelect } from './DirectionSelect';
import { NumericRangeRow } from './NumericRangeRow';
import { ShapeSelect } from './ShapeSelect';
import { ToggleRow } from './ToggleRow';

interface Props {
  enabled: boolean;
  shape: Shape;
  direction: Direction;
  distance: NumericRange;
  speed: NumericRange;
  logging: boolean;
  onEnabledChange: (value: boolean) => void;
  onShapeChange: (shape: Shape) => void;
  onDirectionChange: (direction: Direction) => void;
  onDistanceChange: (update: Partial<NumericRange>) => void;
  onSpeedChange: (update: Partial<NumericRange>) => void;
  onLoggingChange: (value: boolean) => void;
}

export function AdvancedSettingsSection({
  enabled,
  shape,
  direction,
  distance,
  speed,
  logging,
  onEnabledChange,
  onShapeChange,
  onDirectionChange,
  onDistanceChange,
  onSpeedChange,
  onLoggingChange,
}: Props) {
  const { t } = useTranslation();
  const isOnePixel = shape === 'OnePixel';

  return (
    <div className="settings-section">
      <h3 className="section-title">{t('settings.advanced')}</h3>

      <ToggleRow
        label={t('settings.enabled')}
        tooltip={t('tooltip.advanced')}
        value={enabled}
        onChange={onEnabledChange}
      />

      {enabled && (
        <>
          <ShapeSelect value={shape} onChange={onShapeChange} />

          <DirectionSelect value={direction} onChange={onDirectionChange} />

          <NumericRangeRow
            label={t('settings.distance')}
            tooltip={t('tooltip.distance')}
            range={distance}
            onChange={onDistanceChange}
            suffix={t('unit.pixels')}
            min={1}
            max={2000}
            disabled={isOnePixel}
            tooltipMin={t('tooltip.distanceMin')}
            tooltipMax={t('tooltip.distanceMax')}
          />

          <NumericRangeRow
            label={t('settings.speed')}
            tooltip={t('tooltip.speed')}
            range={speed}
            onChange={onSpeedChange}
            suffix={t('unit.seconds')}
            min={1}
            max={60}
            tooltipMin={t('tooltip.speedMin')}
            tooltipMax={t('tooltip.speedMax')}
          />

          <ToggleRow
            label={t('settings.logging')}
            tooltip={t('tooltip.logging')}
            value={logging}
            onChange={onLoggingChange}
          />
        </>
      )}
    </div>
  );
}
