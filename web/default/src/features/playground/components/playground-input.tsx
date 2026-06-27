/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import {
  PaperclipIcon,
  FileIcon,
  ImageIcon,
  ScreenShareIcon,
  CameraIcon,
  GlobeIcon,
  SendIcon,
  SquareIcon,
  BarChartIcon,
  BoxIcon,
  NotepadTextIcon,
  CodeSquareIcon,
  GraduationCapIcon,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  PromptInput,
  PromptInputButton,
  PromptInputFooter,
  PromptInputTextarea,
  PromptInputTools,
  type PromptInputMessage,
} from '@/components/ai-elements/prompt-input'
import { Suggestion, Suggestions } from '@/components/ai-elements/suggestion'
import { ModelGroupSelector } from '@/components/model-group-selector'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

import type { ModelOption, GroupOption } from '../types'

interface PlaygroundInputProps {
  onSubmit: (text: string) => void
  onStop?: () => void
  disabled?: boolean
  isGenerating?: boolean
  models: ModelOption[]
  modelValue: string
  onModelChange: (value: string) => void
  isModelLoading?: boolean
  groups: GroupOption[]
  groupValue: string
  onGroupChange: (value: string) => void
}

const suggestions = [
  { icon: BarChartIcon, text: '分析调用数据', color: '#2563eb' },
  { icon: BoxIcon, text: '生成测试样例', color: '#16a34a' },
  { icon: NotepadTextIcon, text: '总结文本', color: '#f59e0b' },
  { icon: CodeSquareIcon, text: '编写代码', color: '#8b5cf6' },
  { icon: GraduationCapIcon, text: '给出建议', color: '#2563eb' },
  { icon: null, text: '更多' },
]

export function PlaygroundInput({
  onSubmit,
  onStop,
  disabled,
  isGenerating,
  models,
  modelValue,
  onModelChange,
  isModelLoading = false,
  groups,
  groupValue,
  onGroupChange,
}: PlaygroundInputProps) {
  const { t } = useTranslation()
  const [text, setText] = useState('')

  const isModelSelectDisabled =
    disabled || isModelLoading || models.length === 0
  const isGroupSelectDisabled = disabled || groups.length === 0

  const handleSubmit = (message: PromptInputMessage) => {
    if (!message.text?.trim() || disabled) return
    onSubmit(message.text)
    setText('')
  }

  const handleFileAction = (action: string) => {
    toast.info(t('Feature in development'), {
      description: action,
    })
  }

  const handleSuggestionClick = (suggestion: string) => {
    onSubmit(suggestion)
  }

  return (
    <div className='grid shrink-0 gap-3 px-1 md:pb-4'>
      <PromptInput
        groupClassName='rounded-md border-slate-200 bg-white shadow-[0_1px_2px_rgb(15_23_42/0.04)]'
        onSubmit={handleSubmit}
      >
        <PromptInputTextarea
          autoComplete='off'
          autoCorrect='off'
          autoCapitalize='off'
          spellCheck={false}
          className='px-4 text-[13px] md:text-sm'
          disabled={disabled}
          onChange={(event) => setText(event.target.value)}
          placeholder={t('输入测试消息')}
          value={text}
        />

        <PromptInputFooter className='p-2'>
          <PromptInputTools>
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <PromptInputButton
                    className='h-8 rounded-md border border-slate-200 text-[12px] font-medium'
                    disabled={disabled}
                    variant='outline'
                  />
                }
              >
                <PaperclipIcon size={16} />
                <span className='hidden sm:inline'>{t('附件')}</span>
                <span className='sr-only sm:hidden'>{t('附件')}</span>
              </DropdownMenuTrigger>
              <DropdownMenuContent align='start'>
                <DropdownMenuItem
                  onClick={() => handleFileAction('upload-file')}
                >
                  <FileIcon className='mr-2' size={16} />
                  {t('上传文件')}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => handleFileAction('upload-photo')}
                >
                  <ImageIcon className='mr-2' size={16} />
                  {t('上传图片')}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => handleFileAction('take-screenshot')}
                >
                  <ScreenShareIcon className='mr-2' size={16} />
                  {t('截取屏幕')}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => handleFileAction('take-photo')}
                >
                  <CameraIcon className='mr-2' size={16} />
                  {t('拍照')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            <PromptInputButton
              className='h-8 rounded-md border border-slate-200 text-[12px] font-medium'
              disabled={disabled}
              onClick={() => toast.info(t('搜索能力开发中'))}
              variant='outline'
            >
              <GlobeIcon size={16} />
              <span className='hidden sm:inline'>{t('搜索')}</span>
              <span className='sr-only sm:hidden'>{t('搜索')}</span>
            </PromptInputButton>
          </PromptInputTools>

          <div className='flex items-center gap-1.5 md:gap-2'>
            <ModelGroupSelector
              selectedModel={modelValue}
              models={models}
              onModelChange={onModelChange}
              selectedGroup={groupValue}
              groups={groups}
              onGroupChange={onGroupChange}
              disabled={isModelSelectDisabled || isGroupSelectDisabled}
            />

            {isGenerating && onStop ? (
              <PromptInputButton
                className='h-8 rounded-md text-[12px] font-medium'
                onClick={onStop}
                variant='secondary'
              >
                <SquareIcon className='fill-current' size={16} />
                <span className='hidden sm:inline'>{t('停止')}</span>
                <span className='sr-only sm:hidden'>{t('停止')}</span>
              </PromptInputButton>
            ) : (
              <PromptInputButton
                className='h-8 rounded-md text-[12px] font-medium'
                disabled={disabled || !text.trim()}
                type='submit'
                variant='secondary'
              >
                <SendIcon size={16} />
                <span className='hidden sm:inline'>{t('发送')}</span>
                <span className='sr-only sm:hidden'>{t('发送')}</span>
              </PromptInputButton>
            )}
          </div>
        </PromptInputFooter>
      </PromptInput>

      <Suggestions>
        {suggestions.map(({ icon: Icon, text, color }) => (
          <Suggestion
            className={`text-xs font-normal sm:text-sm ${
              text === '更多' ? 'hidden sm:flex' : ''
            }`}
            key={text}
            onClick={() => handleSuggestionClick(text)}
            suggestion={text}
          >
            {Icon && <Icon size={16} style={{ color }} />}
            {text}
          </Suggestion>
        ))}
      </Suggestions>
    </div>
  )
}
