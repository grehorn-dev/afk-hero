import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { ToggleRow } from './ToggleRow';
import type { ActivationMode, WindowInfo } from '../types/settings';
import type { ActivationRuntimeState } from '../types/state';
import { isIntegerDraft, parseIntegerDraft } from '../utils/integerInput';

interface Props {
  enabled: boolean;
  mode: ActivationMode;
  timeout: number;
  targetClass: string;
  targetProcessName: string;
  activationState: ActivationRuntimeState;
  onEnabledChange: (v: boolean) => void;
  onModeChange: (v: ActivationMode) => void;
  onTimeoutChange: (v: number) => void;
  onTargetChange: (processName: string, windowClass: string, title: string) => void;
}

export function WindowActivationSection({
  enabled, mode, timeout,
  onEnabledChange, onModeChange, onTimeoutChange, onTargetChange,
  targetClass, targetProcessName, activationState,
}: Props) {
  const { t } = useTranslation();
  const [windows, setWindows] = useState<WindowInfo[]>([]);
  const [timeoutDraft, setTimeoutDraft] = useState(() => String(timeout));
  const autoCandidate = windows.find((window) => window.autoCandidate);

  useEffect(() => {
    setTimeoutDraft(String(timeout));
  }, [timeout]);

  const formatWindowLabel = useCallback((window: WindowInfo) => {
    const title = window.title.trim() || window.className || window.processName;
    return `${window.processName} - ${title}`;
  }, []);

  const windowValue = useCallback((window: WindowInfo) => {
    return `${window.processName}\u0000${window.className}`;
  }, []);

  const refreshWindows = useCallback(() => {
    window.go.app.App.GetWindowList()
      .then(setWindows)
      .catch(console.error);
  }, []);

  useEffect(() => {
    if (enabled) {
      refreshWindows();
    }
  }, [enabled, refreshWindows]);

  const parsedTimeout = parseIntegerDraft(timeoutDraft);
  const timeoutError = (() => {
    if (timeoutDraft === '') {
      return null;
    }

    if (parsedTimeout === null) {
      return t('validation.invalidNumber');
    }

    if (parsedTimeout < 1) {
      return t('validation.minValue').replace('{min}', '1');
    }

    if (parsedTimeout > 3600) {
      return t('validation.maxValue').replace('{max}', '3600');
    }

    return null;
  })();
  const selectedValue = mode === 'Auto' || !targetProcessName || !targetClass ?
    'auto' :
    `${targetProcessName}\u0000${targetClass}`;
  const activationProgress = Math.max(0, Math.min(1, activationState.progress));
  const autoLabel = (() => {
    if (activationState.targetFound && activationState.processName) {
      const title = activationState.windowTitle.trim() || activationState.processName;
      return `${t('settings.windowActivation.auto')} - ${activationState.processName} - ${title}`;
    }

    if (autoCandidate) {
      return `${t('settings.windowActivation.auto')} - ${formatWindowLabel(autoCandidate)}`;
    }

    return t('settings.windowActivation.auto');
  })();

  const commitTimeout = useCallback(() => {
    if (parsedTimeout === null || parsedTimeout < 1 || parsedTimeout > 3600) {
      setTimeoutDraft(String(timeout));
      return;
    }

    if (parsedTimeout !== timeout) {
      onTimeoutChange(parsedTimeout);
    }
  }, [onTimeoutChange, parsedTimeout, timeout]);

  return (
    <div className="settings-section activation-section">
      <h3 className="section-title">{t('settings.windowActivation')}</h3>

      <ToggleRow
        label={t('settings.windowActivation.enable')}
        tooltip={t('tooltip.windowActivation')}
        value={enabled}
        onChange={onEnabledChange}
      />

      {enabled && (
        <>
          <div className="form-row" title={t('tooltip.targetWindow')}>
            <label className="form-label">{t('settings.windowActivation.window')}</label>
            <div className="window-select-row">
              <select
                className="form-select"
                value={selectedValue}
                onChange={(e) => {
                  const val = e.target.value;
                  if (val === 'auto') {
                    onModeChange('Auto');
                    onTargetChange('', '', '');
                  } else {
                    onModeChange('Manual');
                    const win = windows.find((w) => windowValue(w) === val);
                    onTargetChange(win?.processName || '', win?.className || '', win?.title || '');
                  }
                }}
              >
                <option value="auto">{autoLabel}</option>
                {windows.map((w) => (
                  <option key={`${w.id}-${w.processName}-${w.className}`} value={windowValue(w)}>
                    {formatWindowLabel(w)}
                  </option>
                ))}
              </select>
              <button
                className="refresh-btn"
                onClick={refreshWindows}
                type="button"
                title={t('button.refresh')}
              >
                ↻
              </button>
            </div>
          </div>

          <div className="form-row" title={t('tooltip.activationTimeout')}>
            <label className="form-label">{t('settings.windowActivation.timeout')}</label>
            <div className="range-controls">
              <input
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
                className={`form-input number-input ${timeoutError ? 'error' : ''}`}
                value={timeoutDraft}
                onChange={(e) => {
                  if (isIntegerDraft(e.target.value)) {
                    setTimeoutDraft(e.target.value);
                    const parsed = parseIntegerDraft(e.target.value);
                    if (parsed !== null && parsed >= 1 && parsed <= 3600 && parsed !== timeout) {
                      onTimeoutChange(parsed);
                    }
                  }
                }}
                onBlur={commitTimeout}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    e.currentTarget.blur();
                  }
                }}
                title={timeoutError || t('tooltip.activationTimeout')}
              />
              <span className="suffix-label">{t('unit.seconds')}</span>
            </div>
          </div>

          <div className="form-row activation-progress-row" title={t('tooltip.activationTimeout')}>
            <div className="form-label activation-progress-spacer" aria-hidden="true" />
            <div className={`activation-progress-container${activationState.tracking ? ' is-tracking' : ''}`}>
              <div
                className="activation-progress-bar"
                style={{ width: `${activationProgress * 100}%` }}
              />
            </div>
          </div>
        </>
      )}
    </div>
  );
}
