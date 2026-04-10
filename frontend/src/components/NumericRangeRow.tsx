import { useEffect, useState, type KeyboardEvent } from 'react';
import { useTranslation } from 'react-i18next';
import type { NumericRange, RangeMode } from '../types/settings';
import { RANGE_MODES } from '../types/settings';
import { isIntegerDraft, parseIntegerDraft } from '../utils/integerInput';

interface Props {
  label: string;
  tooltip: string;
  range: NumericRange;
  onChange: (update: Partial<NumericRange>) => void;
  suffix: string;
  min: number;
  max: number;
  disabled?: boolean;
  tooltipMin?: string;
  tooltipMax?: string;
}

export function NumericRangeRow({
  label, tooltip, range, onChange, suffix, min, max, disabled, tooltipMin, tooltipMax,
}: Props) {
  const { t } = useTranslation();
  const isRandom = range.mode === 'Random';
  const [valueDraft, setValueDraft] = useState(() => String(range.value));
  const [minDraft, setMinDraft] = useState(() => String(range.minVal));
  const [maxDraft, setMaxDraft] = useState(() => String(range.maxVal));

  useEffect(() => {
    setValueDraft(String(range.value));
  }, [range.value]);

  useEffect(() => {
    setMinDraft(String(range.minVal));
  }, [range.minVal]);

  useEffect(() => {
    setMaxDraft(String(range.maxVal));
  }, [range.maxVal]);

  const validateValue = (val: number): string | null => {
    if (isNaN(val)) return t('validation.invalidNumber');
    if (val < min) return t('validation.minValue').replace('{min}', String(min));
    if (val > max) return t('validation.maxValue').replace('{max}', String(max));
    return null;
  };

  const validateDraft = (rawVal: string): string | null => {
    if (rawVal === '') return null;

    const num = parseIntegerDraft(rawVal);
    if (num === null) return t('validation.invalidNumber');
    return validateValue(num);
  };

  const parsedMin = parseIntegerDraft(minDraft);
  const parsedMax = parseIntegerDraft(maxDraft);
  const fixedError = !isRandom ? validateDraft(valueDraft) : null;
  const minError = isRandom ? (
    validateDraft(minDraft) || (
      parsedMin !== null && parsedMax !== null && parsedMin > parsedMax
        ? t('validation.minGreaterThanMax')
        : null
    )
  ) : null;
  const maxError = isRandom ? validateDraft(maxDraft) : null;

  const handleDraftChange = (setter: (value: string) => void, rawVal: string) => {
    if (isIntegerDraft(rawVal)) {
      setter(rawVal);
    }
  };

  const commitFixedDraft = (rawVal: string) => {
    const parsed = parseIntegerDraft(rawVal);
    if (parsed === null || validateValue(parsed)) {
      return;
    }

    if (parsed !== range.value) {
      onChange({ value: parsed });
    }
  };

  const commitRandomDraft = (field: 'minVal' | 'maxVal', rawVal: string) => {
    const parsed = parseIntegerDraft(rawVal);
    if (parsed === null || validateValue(parsed)) {
      return;
    }

    const current = field === 'minVal' ? range.minVal : range.maxVal;
    if (parsed !== current) {
      onChange({ [field]: parsed });
    }
  };

  const handleKeyDown = (
    event: KeyboardEvent<HTMLInputElement>,
  ) => {
    if (event.key === 'Enter') {
      event.preventDefault();
      event.currentTarget.blur();
    }
  };

  const commitFixedValue = () => {
    const parsed = parseIntegerDraft(valueDraft);
    if (parsed === null || validateValue(parsed)) {
      setValueDraft(String(range.value));
      return;
    }

    if (parsed !== range.value) {
      onChange({ value: parsed });
    }
  };

  const commitRandomValue = (field: 'minVal' | 'maxVal') => {
    const rawVal = field === 'minVal' ? minDraft : maxDraft;
    const parsed = parseIntegerDraft(rawVal);
    const fallback = field === 'minVal' ? range.minVal : range.maxVal;
    const reset = () => {
      if (field === 'minVal') {
        setMinDraft(String(fallback));
      } else {
        setMaxDraft(String(fallback));
      }
    };

    if (parsed === null || validateValue(parsed)) {
      reset();
      return;
    }

    const current = field === 'minVal' ? range.minVal : range.maxVal;
    if (parsed !== current) {
      onChange({ [field]: parsed });
    }
  };

  return (
    <div className={`form-row ${disabled ? 'disabled' : ''}`} title={tooltip}>
      <label className="form-label">{label}</label>
      <div className="range-controls">
        <select
          className="form-select mode-select"
          value={range.mode}
          onChange={(e) => onChange({ mode: e.target.value as RangeMode })}
          disabled={disabled}
        >
          {RANGE_MODES.map((mode) => (
            <option key={mode} value={mode}>
              {t(`mode.${mode}`)}
            </option>
          ))}
        </select>

        {isRandom ? (
          <>
            <input
              type="text"
              inputMode="numeric"
              pattern="[0-9]*"
              className={`form-input number-input ${minError ? 'error' : ''}`}
              value={minDraft}
              onChange={(e) => {
                handleDraftChange(setMinDraft, e.target.value);
                commitRandomDraft('minVal', e.target.value);
              }}
              onBlur={() => commitRandomValue('minVal')}
              onKeyDown={handleKeyDown}
              disabled={disabled}
              title={minError ? `${minError}\n\n${tooltipMin || ''}` : tooltipMin}
            />
            <input
              type="text"
              inputMode="numeric"
              pattern="[0-9]*"
              className={`form-input number-input ${maxError ? 'error' : ''}`}
              value={maxDraft}
              onChange={(e) => {
                handleDraftChange(setMaxDraft, e.target.value);
                commitRandomDraft('maxVal', e.target.value);
              }}
              onBlur={() => commitRandomValue('maxVal')}
              onKeyDown={handleKeyDown}
              disabled={disabled}
              title={maxError ? `${maxError}\n\n${tooltipMax || ''}` : tooltipMax}
            />
          </>
        ) : (
          <input
            type="text"
            inputMode="numeric"
            pattern="[0-9]*"
            className={`form-input number-input ${fixedError ? 'error' : ''}`}
            value={valueDraft}
            onChange={(e) => {
              handleDraftChange(setValueDraft, e.target.value);
              commitFixedDraft(e.target.value);
            }}
            onBlur={commitFixedValue}
            onKeyDown={handleKeyDown}
            disabled={disabled}
            title={fixedError ? `${fixedError}\n\n${tooltip}` : tooltip}
          />
        )}

        <span className="suffix-label">{suffix}</span>
      </div>
    </div>
  );
}
