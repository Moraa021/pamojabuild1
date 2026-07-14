const STEPS = [
  { id: 'registered',      label: 'Registered',          trigger: 'always'                         },
  { id: 'profile_created', label: 'Profile created',      trigger: 'always'                         },
  { id: 'browsing',        label: 'Task found',           trigger: 'always'                         },
  { id: 'applied',         label: 'Application sent',     trigger: 'applied'                        },
  { id: 'approved',        label: 'Application approved', trigger: 'approved'                       },
  { id: 'working',         label: 'Work in progress',     trigger: 'in_progress'                    },
  { id: 'submitted',       label: 'Evidence submitted',   trigger: 'submission_exists'              },
  { id: 'verifying',       label: 'Under verification',   trigger: 'pending_verification'           },
  { id: 'trustees_review', label: 'Trustee review',       trigger: 'READY_FOR_PAYOUT'               },
  { id: 'cosigning',       label: 'Co-signing underway',  trigger: 'READY_FOR_PAYOUT'               },
  { id: 'broadcast',       label: 'Payment broadcast',    trigger: 'LIQUIDATING'                    },
  { id: 'paid',            label: 'Payment received',     trigger: 'completed'                      },
  { id: 'reputation',      label: 'Reputation updated',   trigger: 'completed'                      },
];

function resolveCompletedSteps({ taskStatus, financialState, hasApplication, applicationApproved, hasSubmission }) {
  const completed = new Set(['registered', 'profile_created', 'browsing']);
  if (hasApplication)           completed.add('applied');
  if (applicationApproved)      completed.add('approved');
  if (taskStatus === 'in_progress' || taskStatus === 'pending_verification' || taskStatus === 'completed') {
    completed.add('working');
  }
  if (hasSubmission)            completed.add('submitted');
  if (taskStatus === 'pending_verification' || taskStatus === 'completed') completed.add('verifying');
  if (financialState === 'READY_FOR_PAYOUT' || taskStatus === 'completed')  {
    completed.add('trustees_review');
    completed.add('cosigning');
  }
  if (financialState === 'LIQUIDATING' || taskStatus === 'completed')       completed.add('broadcast');
  if (taskStatus === 'completed')  { completed.add('paid'); completed.add('reputation'); }
  return completed;
}

export class VerificationTracker {
  #container;
  element;

  constructor(container) {
    this.#container = container;
    this.element    = document.createElement('div');
    this.element.className = 'verification-tracker';
    this.#container.appendChild(this.element);
  }

  update(context) {
    const completed = resolveCompletedSteps(context);
    const stepEls   = STEPS.map((step, i) => {
      const isDone    = completed.has(step.id);
      const isActive  = !isDone && (i === 0 || completed.has(STEPS[i - 1]?.id));
      const cls = isDone ? 'vstep--done' : isActive ? 'vstep--active' : '';
      return `
        <li class="vstep ${cls}" aria-current="${isActive ? 'step' : 'false'}">
          <div class="vstep__indicator" aria-hidden="true">
            ${isDone ? '✓' : i + 1}
          </div>
          <span class="vstep__label">${step.label}</span>
        </li>
      `;
    }).join('');

    this.element.innerHTML = `
      <h4 class="verification-tracker__title">Journey progress</h4>
      <ol class="vstep-list" aria-label="Volunteer workflow steps">
        ${stepEls}
      </ol>
    `;
  }
}
